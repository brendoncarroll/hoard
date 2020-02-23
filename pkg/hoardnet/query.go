package hoardnet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
	"github.com/brendoncarroll/hoard/pkg/tagdb"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultMaxHops  = 1
	DefaultMaxCount = 30
)

type QueryReq = hoardproto.QueryReq
type QueryRes = hoardproto.QueryRes
type Manifest = hoardproto.Manifest

type Queryable interface {
	QueryProtocol(ctx context.Context, q tagdb.Query) ([]*hoardproto.Manifest, error)
}

type QueryService struct {
	local    Queryable
	peers    PeerStore
	s        p2p.SecureAskSwarm
	maxHops  int
	maxCount int
}

func NewQueryService(query Queryable, peerStore PeerStore, s p2p.SecureAskSwarm) *QueryService {
	qs := &QueryService{
		local:    query,
		s:        s,
		maxHops:  DefaultMaxHops,
		maxCount: DefaultMaxCount,
	}
	return qs
}

func (qs *QueryService) QueryRemotes(ctx context.Context, q tagdb.Query) ([]*hoardproto.Manifest, error) {
	if qs.maxCount < q.Limit {
		q.Limit = qs.maxCount
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	req := &QueryReq{
		Limit:    q.Limit,
		Hops:     qs.maxHops,
		Deadline: deadline,
	}

	mfs, err := qs.queryAll(ctx, p2p.ZeroPeerID(), req)
	if err != nil {
		return nil, err
	}
	return mfs, nil
}

func (qs *QueryService) Close() error {
	return nil
}

func (qs *QueryService) queryAll(ctx context.Context, from p2p.PeerID, req *QueryReq) ([]*Manifest, error) {
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	mfsChan := make(chan *Manifest)
	for _, peerID := range qs.peers.ListPeers() {
		peerID := peerID
		go func() {
			res, err := qs.queryRemote(ctx, peerID, req)
			if err != nil {
				log.Error(err)
				return
			}
			for _, mf := range res.Manifests {
				select {
				case <-ctx.Done():
					return
				case mfsChan <- mf:
				}
			}
		}()
	}

	mfs := []*Manifest{}
	for {
		select {
		case <-ctx.Done():
			return mfs, ctx.Err()
		case mf := <-mfsChan:
			mfs = append(mfs, mf)
			if len(mfs) >= req.Limit {
				return mfs, nil
			}
		}
	}
}

func (qs *QueryService) queryRemote(ctx context.Context, id p2p.PeerID, req *QueryReq) (*QueryRes, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	addrs := qs.peers.GetAddrs(id)
	if len(addrs) < 1 {
		return nil, errors.New("no addresses for peer")
	}
	addr := addrs[0]

	resData, err := qs.s.Ask(ctx, addr, reqData)
	if err != nil {
		return nil, err
	}
	res := &QueryRes{}
	if err = json.Unmarshal(resData, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (qs *QueryService) handleAsk(ctx context.Context, m *p2p.Message, w io.Writer) {
	log := log.WithFields(log.Fields{
		"peer_addr": m.Src,
	})

	req := hoardproto.QueryReq{}
	if err := json.Unmarshal(m.Payload, &req); err != nil {
		log.Error("could not parse message")
	}
	limit := req.Limit
	if limit > qs.maxCount {
		limit = qs.maxCount
	}
	hops := req.Hops
	if hops > qs.maxHops {
		hops = qs.maxHops
	}
	if req.Hops < 0 {
		return
	}
	deadline := req.Deadline

	mfs := qs.queryLocal(ctx, req)
	if req.Hops > 0 {
		req2 := &hoardproto.QueryReq{
			Limit:    req.Limit - len(mfs),
			Hops:     hops - 1,
			Deadline: deadline,
		}
		ctx, cf := context.WithDeadline(ctx, deadline)
		id := p2p.LookupPeerID(qs.s, m.Src)
		qs.queryAll(ctx, *id, req2)
		cf()
	}

	res := &QueryRes{
		Manifests: mfs,
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(res); err != nil {
		log.Error(err)
	}
}

func (qs *QueryService) queryLocal(ctx context.Context, req hoardproto.QueryReq) []*Manifest {
	mfs, err := qs.local.QueryProtocol(ctx, req.Query)
	if err != nil {
		log.Error(err)
		return nil
	}
	return mfs
}
