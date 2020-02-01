package hoardnet

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	log "github.com/sirupsen/logrus"
)

type PeerStore interface {
	ListPeers() []p2p.PeerID
	ListAddrs(p2p.PeerID) []p2p.Addr
	Seen(p2p.PeerID, p2p.Addr) error
}

type Healthcheck struct {
	cf        context.CancelFunc
	s         p2p.AskSwarm
	peerStore PeerStore
	period    time.Duration
}

func NewHealthcheck(s p2p.AskSwarm, peerStore PeerStore) *Healthcheck {
	hb := &Healthcheck{
		s:         s,
		peerStore: peerStore,
		period:    3 * time.Second,
	}
	ctx, cf := context.WithCancel(context.Background())
	hb.cf = cf
	go hb.run(ctx)
	return hb
}

func (hb *Healthcheck) run(ctx context.Context) {
	ticker := time.NewTicker(hb.period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range hb.peerStore.ListPeers() {
				for _, addr := range hb.peerStore.ListAddrs(id) {
					if err := hb.checkPeer(ctx, addr); err != nil {
						log.Debug(err)
					}
				}
			}
		}
	}
}

func (hb *Healthcheck) Close() error {
	hb.cf()
	return nil
}

func (hb *Healthcheck) checkPeer(ctx context.Context, addr p2p.Addr) error {
	reqData := []byte("ping")
	_, err := hb.s.Ask(ctx, addr, reqData)
	if err != nil {
		return err
	}
	return nil
}
