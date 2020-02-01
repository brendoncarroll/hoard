package hoardnet

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/pkg/errors"
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
	qs := NewQueryService(query, querySwarm.(p2p.AskSwarm))

	return &HoardNet{
		healthService: hs,
		queryService:  qs,
	}, nil
}

func (hn *HoardNet) Query(ctx context.Context, tags map[string]string) {

}
