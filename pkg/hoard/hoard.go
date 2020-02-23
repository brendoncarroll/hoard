package hoard

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/brendoncarroll/blobcache/pkg/blobcache"
	"github.com/brendoncarroll/blobcache/pkg/blobs"
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/brendoncarroll/go-p2p/s/wlswarm"
	"github.com/brendoncarroll/webfs/pkg/webfsim"
	"github.com/brendoncarroll/webfs/pkg/webref"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"

	"github.com/brendoncarroll/hoard/pkg/boltkv"
	"github.com/brendoncarroll/hoard/pkg/fsbridge"
	"github.com/brendoncarroll/hoard/pkg/hoardnet"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
	"github.com/brendoncarroll/hoard/pkg/tagdb"
	"github.com/brendoncarroll/hoard/pkg/taggers"
)

const bucketManifests = "manifests"

type Node struct {
	params    Params
	localID   p2p.PeerID
	swarm     p2p.SecureAskSwarm
	peerStore *PeerStore
	discover  p2p.DiscoveryService

	fsbridges []fsbridge.Bridge

	hnet *hoardnet.HoardNet

	bcn *blobcache.Node

	db    *bolt.DB
	tagdb *tagdb.TagDB

	suggestedCache sync.Map
}

func New(params *Params) (*Node, error) {
	extSources := []blobs.Getter{}
	bridges := []*fsbridge.Bridge{}
	for _, p := range params.SourcePaths {
		bucketName := "fsbridge"
		kv := boltkv.New(params.DB, bucketName)
		fsbp := fsbridge.Params{
			KV:         kv,
			Path:       p,
			ScanPeriod: 60 * time.Minute,
		}
		b := fsbridge.New(fsbp)
		log.WithFields(log.Fields{
			"path": p,
		}).Info("created fs bridge")

		extSources = append(extSources, b)
		bridges = append(bridges)
	}

	// p2p
	peerStore := newPeerStore(params.DB, params.Swarm)
	swarm := wlswarm.WrapSecureAsk(params.Swarm, peerStore.Contains)
	mux := simplemux.MultiplexSwarm(swarm)

	// blobcache
	cache, err := blobcache.NewBoltKV(params.BlobcacheDB, []byte("data"), params.Capacity)
	if err != nil {
		return nil, err
	}

	bcn, err := blobcache.NewNode(blobcache.Params{
		Mux:             mux,
		PeerStore:       peerStore,
		MetadataDB:      params.BlobcacheDB,
		Cache:           cache,
		ExternalSources: extSources,
	})
	if err != nil {
		return nil, err
	}

	n := &Node{
		params: *params,
		// p2p
		localID:   p2p.NewPeerID(params.Swarm.PublicKey()),
		swarm:     swarm,
		peerStore: peerStore,

		// blobcache
		bcn: bcn,

		// db
		db: params.DB,

		tagdb: tagdb.NewDB(params.DB),
	}
	n.hnet, err = hoardnet.New(mux, n, peerStore)
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

func (n *Node) QueryManifests(ctx context.Context, q tagdb.Query) (*ResultSet, error) {
	tagRes, err := n.tagdb.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	mfs := make([]*Manifest, len(tagRes.IDs))
	for i := range tagRes.IDs {
		mf, err := n.GetManifest(ctx, tagRes.IDs[i])
		if err != nil {
			return nil, err
		}
		mfs[i] = mf
	}

	resultSet := &ResultSet{
		Manifests: mfs,

		Offest: tagRes.Offset,
		Count:  tagRes.Count,
		Total:  tagRes.Total,
	}
	return resultSet, err
}

func (n *Node) QueryProtocol(ctx context.Context, q tagdb.Query) ([]*hoardproto.Manifest, error) {
	tagRes, err := n.tagdb.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	mfs := make([]*hoardproto.Manifest, len(tagRes.IDs))
	for i := range tagRes.IDs {
		mf, err := n.GetManifest(ctx, tagRes.IDs[i])
		if err != nil {
			return nil, err
		}
		pmf := mf.Manifest
		mfs[i] = &pmf
	}
	return mfs, nil
}

func (n *Node) createManifest(ctx context.Context, ref *webref.Ref, pinSetName string) (*Manifest, error) {
	mf := &Manifest{
		Manifest: hoardproto.Manifest{
			WebRef: ref,
		},
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
	if id == 0 {
		return nil, os.ErrNotExist
	}
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

func (n *Node) ListManifests(ctx context.Context, offset, limit int) (*ResultSet, error) {
	// TODO: offset is not supported, because there is no sorting.
	return n.QueryManifests(ctx, tagdb.Query{
		Limit: limit,
	})
}

func (n *Node) GetTag(ctx context.Context, mID uint64, name string) (string, error) {
	return n.tagdb.GetTag(ctx, mID, name)
}

// List Peers returns a list of the ids for every peer in the peer store
func (n *Node) ListPeers(ctx context.Context) ([]p2p.PeerID, error) {
	return n.peerStore.ListPeers(), nil
}

// Get Peer returns the peer info for the peer with the given id
func (n *Node) GetPeer(ctx context.Context, id p2p.PeerID) (*PeerInfo, error) {
	return n.peerStore.GetPeerInfo(id)
}

// PutPeer replaces the pinfo for the peer with ID == pinfo.ID
func (n *Node) PutPeer(ctx context.Context, pinfo *PeerInfo) error {
	return n.peerStore.PutPeerInfo(pinfo)
}

// DeletePeer deletes the peer's info from the node
func (n *Node) DeletePeer(ctx context.Context, id p2p.PeerID) error {
	return n.peerStore.DeletePeer(id)
}

func (n *Node) SuggestTags(ctx context.Context, id uint64) (taggers.TagSet, error) {
	mf, err := n.GetManifest(ctx, id)
	if err != nil {
		return nil, err
	}
	ref := mf.WebRef

	v, exists := n.suggestedCache.Load(ref.String())
	if exists {
		return v.(taggers.TagSet), nil
	}
	rc, err := n.openFile(ctx, *ref, "")
	if err != nil {
		return nil, err
	}
	tags := make(taggers.TagSet)
	if err := taggers.SuggestTags(rc, tags); err != nil {
		return nil, err
	}

	n.suggestedCache.Store(ref.String(), tags)
	return tags, nil
}

func (n *Node) Close() error {
	errs := []error{
		n.db.Close(),
		n.hnet.Close(),
	}
	found := false
	for _, err := range errs {
		if err != nil {
			found = true
		}
	}
	if found {
		return fmt.Errorf("errors closing: %v", errs)
	}
	return nil
}

func (n *Node) getUIPath() string {
	return n.params.UIPath
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
