package hoard

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/brendoncarroll/blobcache/pkg/blobcache"
	"github.com/brendoncarroll/blobcache/pkg/blobs"
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/brendoncarroll/hoard/pkg/hoardnet"
	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/webfs/pkg/webfsim"
	"github.com/brendoncarroll/webfs/pkg/webref"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const bucketManifests = "manifests"

type Node struct {
	localID   p2p.PeerID
	swarm     p2p.SecureAskSwarm
	mux       simplemux.Muxer
	peerStore *PeerStore
	discover  p2p.DiscoveryService

	hnet *hoardnet.HoardNet

	bcn *blobcache.Node

	db    *bolt.DB
	tagdb *TagDB

	suggestedCache sync.Map
}

func New(params *Params) (*Node, error) {
	extSources := []blobs.Getter{}
	// for _, p := range params.SourcePaths {
	// 	spec := fsbridge.Spec
	// 	b := fsbridge.New(spec, nil) //TODO
	// 	extSources = append(extSources)
	// }

	cache, err := blobcache.NewBoltKV(params.BlobcacheDB, []byte("data"), params.Capacity)
	if err != nil {
		return nil, err
	}
	bcp := blobcache.Params{
		MetadataDB:      params.BlobcacheDB,
		Cache:           cache,
		ExternalSources: extSources,
	}
	bcn, err := blobcache.NewNode(bcp)
	if err != nil {
		return nil, err
	}

	n := &Node{
		// p2p
		localID:   p2p.NewPeerID(params.Swarm.PublicKey()),
		swarm:     params.Swarm,
		mux:       params.Mux,
		peerStore: newPeerStore(params.DB),

		// blobcache
		bcn: bcn,

		// db
		db: params.DB,

		tagdb: NewTagDB(params.DB),
	}
	n.hnet, err = hoardnet.New(params.Mux, n, n.peerStore)
	if err != nil {
		return nil, err
	}

	return n, nil
}

// AddFile imports and creates a manifest for the file at p
func (n *Node) AddFile(ctx context.Context, p string) (*Manifest, error) {
	log.Println("adding file", p)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	pinSetName := n.genPinSetName()
	if err := n.bcn.CreatePinSet(ctx, pinSetName); err != nil {
		return nil, err
	}
	s := makeStore(n.bcn, pinSetName)
	wf, err := webfsim.FileFromReader(ctx, s, f)
	if err != nil {
		return nil, err
	}
	o := &webfsim.Object{
		Value: &webfsim.Object_File{wf},
	}
	ctx = webref.SetCodecCtx(ctx, webref.CodecProtobuf)
	ref, err := webref.EncodeAndPost(ctx, s, o)
	if err != nil {
		return nil, err
	}

	mf, err := n.createManifest(ctx, ref, pinSetName)
	if err != nil {
		return nil, err
	}

	for k, v := range map[string]string{
		"filename":  filepath.Base(p),
		"extension": filepath.Ext(p),
	} {
		if err := n.tagdb.PutTag(ctx, mf.ID, k, v); err != nil {
			return nil, err
		}
	}

	return n.GetManifest(ctx, mf.ID)
}

// AddAllFiles calls AddFile for each file with a path below p
func (n *Node) AddAllFiles(ctx context.Context, p string) error {
	finfo, err := os.Stat(p)
	if err != nil {
		return err
	}
	if finfo.IsDir() {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		finfos, err := f.Readdir(0)
		if err != nil {
			return err
		}
		for _, finfo := range finfos {
			if err := n.AddAllFiles(ctx, filepath.Join(p, finfo.Name())); err != nil {
				return err
			}
		}
	} else {
		_, err := n.AddFile(ctx, p)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddDir adds a directory as a single manifest
func (n *Node) AddDir(ctx context.Context, p string) error {
	panic("not implemented")
}

// PutTag associates a tag with the manifest associated with id
func (n *Node) PutTag(ctx context.Context, id uint64, key, value string) (*Manifest, error) {
	if err := n.tagdb.PutTag(ctx, id, key, value); err != nil {
		return nil, err
	}
	return n.GetManifest(ctx, id)
}

// DeleteTag removes a tag from the manifest associated with id
func (n *Node) DeleteTag(ctx context.Context, id uint64, key string) (*Manifest, error) {
	if err := n.tagdb.DeleteTag(ctx, id, key); err != nil {
		return nil, err
	}
	return n.GetManifest(ctx, id)
}

func (n *Node) GetData(ctx context.Context, id uint64, p string) (io.ReadSeeker, error) {
	mf, err := n.GetManifest(ctx, id)
	if err != nil {
		return nil, err
	}
	return n.openFile(ctx, *mf.WebRef, p)
}

func (n *Node) openFile(ctx context.Context, r webref.Ref, p string) (io.ReadSeeker, error) {
	o := &webfsim.Object{}
	s := makeStore(n.bcn, "")
	if err := webref.GetAndDecode(ctx, s, r, o); err != nil {
		return nil, err
	}
	if o.GetFile() != nil {
		fr := webfsim.NewFileReader(s, o.GetFile())
		return fr, nil
	}
	return nil, errors.New("cannot get data from webfs object")
}

func (n *Node) QueryManifests(ctx context.Context, tags taggers.TagSet, limit int) ([]uint64, error) {
	// TODO: implement filtering
	resultSet := []uint64{}
	err := n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketManifests))

		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			id := bytesToID(k)
			resultSet = append(resultSet, id)
		}
		return nil
	})
	return resultSet, err
}

func (n *Node) createManifest(ctx context.Context, ref *webref.Ref, pinSetName string) (*Manifest, error) {
	mf := &Manifest{
		WebRef:     ref,
		PinSetName: pinSetName,
	}

	err := n.db.Update(func(tx *bolt.Tx) error {
		mb, err := tx.CreateBucketIfNotExists([]byte(bucketManifests))
		if err != nil {
			return err
		}
		i, err := mb.NextSequence()
		if err != nil {
			return err
		}
		mf.ID = i

		value, err := json.Marshal(mf)
		if err != nil {
			return err
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, i)
		if err := mb.Put(key, value); err != nil {
			return err
		}

		mf.ID = i
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Println("created manifest", "id:", mf.ID)
	return mf, nil
}

func (n *Node) GetManifest(ctx context.Context, id uint64) (*Manifest, error) {
	mf := &Manifest{}
	err := n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketManifests))
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, id)
		value := b.Get(key)
		return json.Unmarshal(value, &mf)
	})
	if err != nil {
		return nil, err
	}
	mf.ID = id

	// tags
	tags, err := n.tagdb.AllTagsFor(ctx, id)
	if err != nil {
		return nil, err
	}
	mf.Tags = tags
	// suggested tags
	// this can be slow
	mf.SuggestedTags = n.suggestTags(ctx, mf.WebRef)

	// pinset
	pinSet, err := n.bcn.GetPinSet(ctx, mf.PinSetName)
	if err != nil {
		return nil, err
	}
	mf.BlobCount = pinSet.Count
	if pinSet.Root != blobs.ZeroID() {
		mf.PinSetRoot = &pinSet.Root
	}
	mf.Peer = n.localID

	return mf, nil
}

func (n *Node) GetTag(ctx context.Context, mID uint64, name string) (string, error) {
	return n.tagdb.GetTag(ctx, mID, name)
}

func (n *Node) Serve(ctx context.Context, laddr string) error {
	log.Debug("Serving UI on ", laddr, "...")
	hapi := newHTTPAPI(n)
	return http.ListenAndServe(laddr, hapi)
}

func (n *Node) suggestTags(ctx context.Context, ref *webref.Ref) taggers.TagSet {
	v, exists := n.suggestedCache.Load(ref.String())
	if exists {
		return v.(taggers.TagSet)
	}
	rc, err := n.openFile(ctx, *ref, "")
	if err != nil {
		log.Println(err)
		return nil
	}
	tags := make(taggers.TagSet)
	if err := taggers.SuggestTags(rc, tags); err != nil {
		log.Println(err)
		return nil
	}
	n.suggestedCache.Store(ref.String(), tags)
	return tags
}

func (n *Node) genPinSetName() string {
	x := time.Now().UnixNano()
	return fmt.Sprintf("hoard-%d", x)
}

func bytesToID(buf []byte) uint64 {
	return binary.BigEndian.Uint64(buf)
}

func idToBytes(x uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], x)
	return buf[:]
}
