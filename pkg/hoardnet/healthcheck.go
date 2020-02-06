package hoardnet

import (
	"context"
	"io"
	"time"

	"github.com/brendoncarroll/go-p2p"
	log "github.com/sirupsen/logrus"
)

type PeerStore interface {
	ListPeers() []p2p.PeerID
	ListAddrs(p2p.PeerID) []string
	Seen(p2p.PeerID, string) error
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
		period:    10 * time.Second,
	}
	s.OnAsk(hb.handleAsk)

	ctx, cf := context.WithCancel(context.Background())
	hb.cf = cf
	go hb.run(ctx)

	return hb
}

func (hb *Healthcheck) Close() error {
	hb.cf()
	return nil
}

func (hb *Healthcheck) run(ctx context.Context) {
	log.Info("Healthcheck service starting")
	ticker := time.NewTicker(hb.period)
	defer ticker.Stop()

	hb.checkAll(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Println("Healthcheck service exiting")
			return
		case <-ticker.C:
			hb.checkAll(ctx)
		}
	}
}

func (hb *Healthcheck) checkAll(ctx context.Context) {
	ctx, cf := context.WithTimeout(ctx, hb.period/2)
	defer cf()
	for _, id := range hb.peerStore.ListPeers() {
		for _, addrStr := range hb.peerStore.ListAddrs(id) {
			addr := p2p.NewAddrOfType(hb.s)
			if err := addr.UnmarshalText([]byte(addrStr)); err != nil {
				log.Error("could not parse addr:", addrStr)
				continue
			}

			go func() {
				if err := hb.checkPeer(ctx, addr); err != nil {
					log.Debug(err)
				} else {
					addrText, _ := addr.MarshalText()
					hb.peerStore.Seen(id, string(addrText))
				}
			}()
		}
	}
}

func (hb *Healthcheck) checkPeer(ctx context.Context, addr p2p.Addr) error {
	reqData := []byte("ping")
	log.Trace("healthcheck sending ping")
	_, err := hb.s.Ask(ctx, addr, reqData)
	if err != nil {
		return err
	}
	log.Trace("healthcheck received pong")
	return nil
}

func (hb *Healthcheck) handleAsk(ctx context.Context, msg *p2p.Message, w io.Writer) {
	res := []byte("pong")
	w.Write(res)
}
