package hoard

import (
	"github.com/brendoncarroll/go-p2p"
	bolt "go.etcd.io/bbolt"
)

type Status struct {
	LocalID       p2p.PeerID
	ManifestCount uint64
	Addrs         []string
	Peers         []*PeerInfo
}

func (n *Node) Status() Status {
	count := uint64(0)
	n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketManifests))
		count = uint64(b.Stats().KeyN)
		return nil
	})

	addrs := []string{}
	for _, paddr := range n.swarm.LocalAddrs() {
		data, _ := paddr.MarshalText()
		addrs = append(addrs, string(data))
	}

	peerInfos := []*PeerInfo{}
	for _, id := range n.peerStore.ListPeers() {
		pinfo := n.peerStore.GetPeerInfo(id)
		peerInfos = append(peerInfos, pinfo)
	}

	return Status{
		LocalID:       n.localID,
		ManifestCount: count,
		Addrs:         addrs,
		Peers:         peerInfos,
	}
}
