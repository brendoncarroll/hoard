package hoardnet

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
	"github.com/brendoncarroll/hoard/pkg/tagdb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	HealthChannel = "hoard/health-v0"
	QueryChannel  = "hoard/query-v0"
)

type HoardNet struct {
	queryService  *QueryService
	healthService *Healthcheck
}

func New(mux simplemux.Muxer, query Queryable, peerStore PeerStore) (*HoardNet, error) {
	healthSwarm, err := mux.OpenChannel(HealthChannel)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open health channel")
	}
	hs := NewHealthcheck(healthSwarm.(p2p.AskSwarm), peerStore)

	querySwarm, err := mux.OpenChannel(QueryChannel)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open search channel on mux")
	}
	qs := NewQueryService(query, peerStore, querySwarm.(p2p.SecureAskSwarm))

	return &HoardNet{
		healthService: hs,
		queryService:  qs,
	}, nil
}

func (hn *HoardNet) QueryPeers(ctx context.Context, q tagdb.Query) ([]*hoardproto.Manifest, error) {
	mfs, err := hn.queryService.QueryRemotes(ctx, q)
	return mfs, err
}

func (hn *HoardNet) Close() error {
	srvs := []interface {
		Close() error
	}{
		hn.queryService,
		hn.healthService,
	}
	for _, s := range srvs {
		if err := s.Close(); err != nil {
			log.Error(err)
		}
	}
	return nil
}
