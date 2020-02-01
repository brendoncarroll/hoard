package hoardnet

import (
	"context"
	"io"
	"log"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
	"github.com/brendoncarroll/hoard/pkg/taggers"
)

type Queryable interface {
	QueryManifests(ctx context.Context, tags taggers.TagSet, limit int) ([]uint64, error)
	GetManifest(ctx context.Context, id uint64) (*hoardproto.Manifest, error)
}

type QueryService struct {
	s p2p.AskSwarm
}

func NewQueryService(query Queryable, s p2p.AskSwarm) *QueryService {
	qs := &QueryService{s: s}
	return qs
}

func (qs *QueryService) handleAsk(ctx context.Context, m *p2p.Message, w io.Writer) {
	log.Println("MSG:", m)
}
