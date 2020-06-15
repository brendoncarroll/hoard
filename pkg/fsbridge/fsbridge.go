package fsbridge

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/blobcache/blobcache/pkg/blobs"
	cbor "github.com/brianolson/cbor_go"
	log "github.com/sirupsen/logrus"
)

var _ blobs.Getter = &Bridge{}

const MaxSize = blobs.MaxSize

type PutGet interface {
	Put(k, v []byte) error
	Get(k []byte) ([]byte, error)
}

type Params struct {
	KV         PutGet
	Path       string
	ScanPeriod time.Duration
}

type Entry struct {
	Path   string `cbor:"p"`
	Offset int64  `cbor:"o"`
	Length int    `cbor:"l"`
}

type Bridge struct {
	path       string
	kv         PutGet
	transform  Transform
	scanPeriod time.Duration
	cf         context.CancelFunc
}

func New(params Params) *Bridge {
	p, err := filepath.Abs(params.Path)
	if err != nil {
		panic(err)
	}
	ctx, cf := context.WithCancel(context.Background())
	b := &Bridge{
		kv:         params.KV,
		path:       p,
		transform:  WebFSTransform,
		cf:         cf,
		scanPeriod: params.ScanPeriod,
	}
	if b.scanPeriod > 0 {
		go b.run(ctx)
	}
	return b
}

func (b *Bridge) run(ctx context.Context) error {
	if err := b.Index(ctx, b.path); err != nil {
		log.Error(err)
	}

	ticker := time.NewTicker(b.scanPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := b.Index(ctx, b.path); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *Bridge) Close() error {
	b.cf()
	return nil
}

func (b *Bridge) Index(ctx context.Context, p string) (err error) {
	log.Info("fs_bridge: start indexing", p)
	if b.indexPath(ctx, p); err != nil {
		return err
	}
	log.Info("fs_bridge: done indexing", p)
	return nil
}

func (b *Bridge) GetF(ctx context.Context, id blobs.ID, fn func([]byte) error) error {
	ent, err := b.getEntry(id)
	if err != nil {
		return err
	}
	if ent == nil {
		return blobs.ErrNotFound
	}

	f, err := os.Open(ent.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(ent.Offset, io.SeekStart); err != nil {
		return err
	}

	r := io.LimitReader(f, int64(ent.Length))
	data := make([]byte, ent.Length)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return err
	}
	return fn(data)
}

func (b *Bridge) Exists(ctx context.Context, id blobs.ID) (bool, error) {
	x, err := b.kv.Get(id[:])
	if err != nil {
		return false, err
	}
	return len(x) > 0, nil
}

func (b *Bridge) List(ctx context.Context, prefix []byte, ids []blobs.ID) (n int, err error) {
	return 0, nil
}

func (b *Bridge) indexPath(ctx context.Context, p string) error {
	finfo, err := os.Stat(p)
	if err != nil {
		return err
	}
	if finfo.IsDir() {
		return b.indexDir(ctx, p)
	}

	t, err := b.whenModified(ctx, p)
	if err != nil {
		log.Error(err)
	}
	if t != nil && !finfo.ModTime().After(*t) {
		log.Trace("skipping file ", p)
		return nil
	}
	return b.indexFile(ctx, p)
}

func (b *Bridge) indexDir(ctx context.Context, p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	finfos, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	for _, finfo := range finfos {
		p2 := filepath.Join(p, finfo.Name())
		if err := b.indexPath(ctx, p2); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bridge) indexFile(ctx context.Context, p string) error {
	log.Trace("fsbridge: indexing file: ", p)
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	chunker := NewFixedSizeChunker(f, MaxSize)
	err = chunker.ForEachChunk(func(chunk Chunk) error {
		b.transform.TransformInPlace(chunk.Data)
		id := blobs.Hash(chunk.Data)
		e := Entry{
			Offset: chunk.Offset,
			Path:   p,
			Length: len(chunk.Data),
		}
		if err := b.putEntry(ctx, id, e); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	finfo, err := f.Stat()
	if err != nil {
		return err
	}
	modTime := finfo.ModTime()
	if err := b.markModified(ctx, p, modTime); err != nil {
		return err
	}
	return nil
}

func (b *Bridge) markModified(ctx context.Context, p string, mtime time.Time) error {
	v, err := json.Marshal(mtime)
	if err != nil {
		panic(err)
	}
	key := keyForModified(p)
	return b.kv.Put(key, v)
}

func (b *Bridge) whenModified(ctx context.Context, p string) (*time.Time, error) {
	key := keyForModified(p)
	v, err := b.kv.Get(key)
	if err != nil {
		return nil, err
	}
	if len(v) < 1 {
		return nil, nil
	}
	t := &time.Time{}
	if err := json.Unmarshal(v, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (b *Bridge) putEntry(ctx context.Context, id blobs.ID, e Entry) error {
	value, err := cbor.Dumps(e)
	if err != nil {
		return err
	}
	key := keyForEntry(id)
	return b.kv.Put(key, value)
}

func (b *Bridge) getEntry(id blobs.ID) (*Entry, error) {
	key := keyForEntry(id)
	value, err := b.kv.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	e := &Entry{}
	if err := cbor.Loads(value, e); err != nil {
		return nil, err
	}
	return e, nil
}

func keyForEntry(id blobs.ID) []byte {
	return append([]byte{'e'}, id[:]...)
}

func keyForModified(p string) []byte {
	return append([]byte{'m'}, []byte(p)...)
}
