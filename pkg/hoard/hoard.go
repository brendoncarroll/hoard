package hoard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/blobcache/glfs"
	"github.com/brendoncarroll/go-state"
	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/go-state/cells"
	"github.com/gotvc/got/pkg/gotkv"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/brendoncarroll/hoard/pkg/hcorpus"
	"github.com/brendoncarroll/hoard/pkg/hexpr"
	"github.com/brendoncarroll/hoard/pkg/hindex"
	"github.com/brendoncarroll/hoard/pkg/labels"
)

type (
	ID     = hcorpus.ID
	Expr   = hexpr.Expr
	IDSpan = cadata.Span
)

type State struct {
	Corpus  hcorpus.Root           `json:"corpus"`
	Indexes map[string]hindex.Root `json:"indexes"`
}

type Volume struct {
	Cell   cells.Cell
	Corpus cadata.Store
	Index  cadata.Store

	GLFS cadata.Store
}

type Indexer func(ctx context.Context, e hexpr.Expr, cv hexpr.Value) ([]labels.Pair, error)

type Params struct {
	Volume   Volume
	Indexers map[string]Indexer
}

type Hoard struct {
	vol      Volume
	indexers map[string]Indexer

	hindex  *hindex.Operator
	hcorpus *hcorpus.Operator
}

func New(params Params) *Hoard {
	return &Hoard{
		vol:     params.Volume,
		hindex:  hindex.New(),
		hcorpus: hcorpus.New(),
	}
}

func (h *Hoard) Add(ctx context.Context, r io.Reader) (*ID, error) {
	var ret *ID
	ref, err := glfs.PostBlob(ctx, h.vol.GLFS, r)
	if err != nil {
		return nil, err
	}
	e := hexpr.NewGLFS(*ref)
	rc2, err := glfs.GetBlob(ctx, h.vol.GLFS, *ref)
	if err != nil {
		return nil, err
	}
	v := hexpr.Value{Data: rc2}
	if err := h.update(ctx, func(s *State) (*State, error) {
		var croot *hcorpus.Root
		// init
		if s != nil {
			croot = &s.Corpus
		} else {
			var err error
			if croot, err = h.hcorpus.NewEmpty(ctx, h.vol.Corpus); err != nil {
				return nil, err
			}
		}
		// add to corpus
		fp, croot2, err := h.hcorpus.Post(ctx, h.vol.Corpus, *croot, hexpr.Marshal(e))
		if err != nil {
			return nil, err
		}
		ret = &fp
		// add to indexes
		iroots := make(map[string]hindex.Root)
		for iname := range h.indexers {
			if root, exists := s.Indexes[iname]; exists {
				iroots[iname] = root
			} else {
				r, err := h.hindex.NewEmpty(ctx, h.vol.Index)
				if err != nil {
					return nil, err
				}
				iroots[iname] = *r
			}
		}
		eg := errgroup.Group{}
		for iname, idxer := range h.indexers {
			iname := iname
			idxer := idxer
			eg.Go(func() error {
				tags, err := idxer(ctx, e, v)
				if err != nil {
					return err
				}
				root, err := h.hindex.AddTags(ctx, h.vol.Index, iroots[iname], fp, tags)
				if err != nil {
					return err
				}
				iroots[iname] = *root
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return nil, err
		}
		return &State{
			Corpus:  *croot2,
			Indexes: iroots,
		}, nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (h *Hoard) NewReaderAt(ctx context.Context, id ID) (io.ReaderAt, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("no object with that id")
	}
	ev := h.newEvaluator(ctx, x)
	v, err := ev.EvalID(ctx, id)
	if err != nil {
		return nil, err
	}
	return v.Data, nil
}

func (h *Hoard) NewReader(ctx context.Context, id ID) (io.ReadSeeker, error) {
	r, err := h.NewReaderAt(ctx, id)
	if err != nil {
		return nil, err
	}
	if rs, ok := r.(io.ReadSeeker); ok {
		return rs, nil
	}
	return io.NewSectionReader(r, 0, math.MaxInt64), nil
}

func (h *Hoard) newEvaluator(ctx context.Context, x *State) *hexpr.Evaluator {
	return &hexpr.Evaluator{
		GetExpr: func(ctx context.Context, id ID) (*hexpr.Expr, error) {
			data, err := h.hcorpus.Get(ctx, h.vol.Corpus, x.Corpus, id)
			if err != nil {
				return nil, err
			}
			return hexpr.ParseExpr(data)
		},
		OpenGLFS: func(ref glfs.Ref) (io.ReaderAt, error) {
			return glfs.GetBlob(ctx, h.vol.GLFS, ref)
		},
	}
}

func (h *Hoard) GetLabels(ctx context.Context, id ID, indexName string) ([]labels.Pair, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("no object with that id")
	}
	iroot, exists := x.Indexes[indexName]
	if !exists {
		return nil, errors.Errorf("index not found: %q", indexName)
	}
	return h.hindex.GetTags(ctx, h.vol.Index, iroot, id)
}

func (h *Hoard) ListIndexes(ctx context.Context) ([]string, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	keys := maps.Keys(x.Indexes)
	slices.Sort(keys)
	return keys, nil
}

func (h *Hoard) ForEachExpr(ctx context.Context, span state.Span[cadata.ID], fn func(id ID, e hexpr.Expr) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	if x == nil {
		return nil
	}
	return h.hcorpus.ForEach(ctx, h.vol.Corpus, x.Corpus, gotkv.TotalSpan(), func(id hcorpus.ID) error {
		v, err := h.hcorpus.Get(ctx, h.vol.Corpus, x.Corpus, id)
		if err != nil {
			return err
		}
		e, err := hexpr.ParseExpr(v)
		if err != nil {
			return err
		}
		return fn(id, *e)
	})
}

func (h *Hoard) ListIDs(ctx context.Context, span state.Span[cadata.ID]) (ret []ID, _ error) {
	err := h.ForEachExpr(ctx, span, func(id ID, e Expr) error {
		ret = append(ret, id)
		return nil
	})
	return ret, err
}

func (h *Hoard) ForEachKey(ctx context.Context, index string, fn func(string) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	if index != "" {
		iroot, exists := x.Indexes[index]
		if !exists {
			return fmt.Errorf("index does not exist %v", index)
		}
		return h.hindex.ForEachKey(ctx, h.vol.Index, iroot, fn)
	}
	for _, iroot := range x.Indexes {
		if err := h.hindex.ForEachKey(ctx, h.vol.Index, iroot, fn); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hoard) ForEachValue(ctx context.Context, index, tagKey string, fn func([]byte) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	if index != "" {
		iroot, exists := x.Indexes[index]
		if !exists {
			return fmt.Errorf("index does not exist %v", index)
		}
		return h.hindex.ForEachValue(ctx, h.vol.Index, iroot, tagKey, fn)
	}
	for _, iroot := range x.Indexes {
		if err := h.hindex.ForEachValue(ctx, h.vol.Index, iroot, tagKey, fn); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hoard) Search(ctx context.Context, query labels.Query) (ret []ID, _ error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	for _, iroot := range x.Indexes {
		res, err := h.hindex.Search(ctx, h.vol.Index, iroot, query)
		if err != nil {
			return nil, err
		}
		ret = append(ret, res.IDs...)
	}
	return ret, nil
}

func (h *Hoard) update(ctx context.Context, fn func(*State) (*State, error)) error {
	return cells.Apply(ctx, h.vol.Cell, func(data []byte) ([]byte, error) {
		var x *State
		if len(data) > 0 {
			x = &State{}
			if err := json.Unmarshal(data, x); err != nil {
				return nil, err
			}
		}
		y, err := fn(x)
		if err != nil {
			return nil, err
		}
		if y == nil {
			return nil, nil
		}
		return json.Marshal(y)
	})
}

func (h *Hoard) get(ctx context.Context) (*State, error) {
	var x State
	if err := getJSON(ctx, h.vol.Cell, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func getJSON(ctx context.Context, cell cells.Cell, x interface{}) error {
	data, err := cells.GetBytes(ctx, cell)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &x)
}
