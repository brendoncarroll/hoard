package hoard

import (
	"encoding/json"
	"time"

	"github.com/blobcache/blobcache/pkg/blobnet/peers"
	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var _ peers.PeerStore = &PeerStore{}

const bucketPeers = "peers"

type PeerInfo struct {
	ID       p2p.PeerID `json:"id"`
	Nickname string     `json:"nickname"`

	Trust           int64    `json:"trust"`
	DiscoveryTokens []string `json:"discovery_tokens"`
	StaticAddrs     []string `json:"static_addrs"`

	SeenAt map[string]time.Time `json:"seen_at"`
}

type PeerStore struct {
	s  p2p.Swarm
	db *bolt.DB
}

func newPeerStore(db *bolt.DB, s p2p.Swarm) *PeerStore {
	return &PeerStore{db: db}
}

func (ps *PeerStore) PutPeerInfo(pinfo *PeerInfo) error {
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

func (ps *PeerStore) GetPeerInfo(id p2p.PeerID) (*PeerInfo, error) {
	var data []byte
	err := ps.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketPeers))
		value := b.Get(id[:])
		data = append([]byte{}, value...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	pinfo := &PeerInfo{}
	if err := json.Unmarshal(data, pinfo); err != nil {
		return nil, errors.Wrap(err, "hello ")
	}
	return pinfo, nil
}

func (ps *PeerStore) DeletePeer(id p2p.PeerID) error {
	err := ps.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketPeers))
		b.Delete(id[:])
		return nil
	})
	return err
}

func (ps *PeerStore) update(id p2p.PeerID, fn func(x *PeerInfo) PeerInfo) error {
	err := ps.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPeers))
		if err != nil {
			return err
		}
		valueCurrent := b.Get(id[:])

		x := &PeerInfo{}
		if err := json.Unmarshal(valueCurrent, &x); err != nil {
			return err
		}
		y := fn(x)
		valueNext, err := json.Marshal(y)
		if err != nil {
			return err
		}
		return b.Put(id[:], valueNext)
	})
	return err
}

func (ps *PeerStore) AddStaticAddr(id p2p.PeerID, addr string) error {
	return ps.update(id, func(x *PeerInfo) PeerInfo {
		y := *x
		y.StaticAddrs = append(y.StaticAddrs, addr)
		return y
	})
}

func (ps *PeerStore) Seen(id p2p.PeerID, addr p2p.Addr) error {
	data, err := addr.MarshalText()
	if err != nil {
		return err
	}
	addrStr := string(data)
	return ps.update(id, func(x *PeerInfo) PeerInfo {
		y := *x
		y.SeenAt[addrStr] = time.Now()
		return y
	})
}

func (ps *PeerStore) Contains(id p2p.PeerID) bool {
	_, err := ps.GetPeerInfo(id)
	if err != nil {
		return false
	}
	return true
}

func (ps *PeerStore) ListPeers() []p2p.PeerID {
	ids := []p2p.PeerID{}
	err := ps.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketPeers))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v == nil {
				continue
			}
			peerInfo := &PeerInfo{}
			if err := json.Unmarshal(v, peerInfo); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		err = errors.Wrap(err, "error retrieving peers from store")
		log.Error(err)
		return []p2p.PeerID{}
	}

	return ids
}

func (ps *PeerStore) GetAddrs(id p2p.PeerID) []p2p.Addr {
	pinfo, err := ps.GetPeerInfo(id)
	if err != nil {
		log.Error(err)
		return nil
	}
	addrs := []p2p.Addr{}
	for _, addrStr := range pinfo.StaticAddrs {
		addr, err := ps.s.ParseAddr([]byte(addrStr))
		if err != nil {
			log.Error(err)
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

func (ps *PeerStore) TrustFor(id p2p.PeerID) (int64, error) {
	pinfo, err := ps.GetPeerInfo(id)
	if err != nil {
		return 0, err
	}
	return pinfo.Trust, nil
}
