package hoard

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/brendoncarroll/blobcache/pkg/blobcache"
	"github.com/brendoncarroll/go-p2p"
	bolt "go.etcd.io/bbolt"
)

var _ blobcache.PeerStore = &PeerStore{}

const bucketPeers = "peers"

type PeerInfo struct {
	ID p2p.PeerID `json:"id"`

	Trust           int64    `json:"trust"`
	DiscoveryTokens []string `json:"discovery_tokens"`
	Addrs           []string `json:"addrs"`

	SeenAt map[string]time.Time `json:"seen_at"`
}

type PeerStore struct {
	db *bolt.DB
}

func newPeerStore(db *bolt.DB) *PeerStore {
	return &PeerStore{db: db}
}

func (ps *PeerStore) PutPeer(pinfo *PeerInfo) error {
	data, _ := json.Marshal(pinfo)
	err := ps.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPeers))
		if err != nil {
			return err
		}
		return b.Put(pinfo.ID[:], data)
	})
	return err
}

func (ps *PeerStore) AddAddr(id p2p.PeerID, addr string) error {
	return nil
}

func (ps *PeerStore) Seen(id p2p.PeerID, addr p2p.Addr) error {
	return nil
}

func (ps *PeerStore) GetPeerInfo(id p2p.PeerID) *PeerInfo {
	return nil
}

func (ps *PeerStore) Contains(id p2p.PeerID) bool {
	return false
}

func (ps *PeerStore) ListPeers() []p2p.PeerID {
	return nil
}

func (ps *PeerStore) ListAddrs(id p2p.PeerID) []p2p.Addr {
	return nil
}

func (ps *PeerStore) TrustFor(id p2p.PeerID) (int64, error) {
	return 0, errors.New("not trusted")
}
