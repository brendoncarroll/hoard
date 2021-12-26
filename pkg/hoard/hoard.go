package hoard

import (
	"context"
	"encoding/json"
	"io"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/go-state/cells"
	"github.com/brendoncarroll/hoard/pkg/hcorpus"
	"github.com/brendoncarroll/hoard/pkg/hindex"
	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/hoard/pkg/tagging"
	"github.com/pkg/errors"
)

type State struct {
	Corpus hcorpus.Root `json:"corpus"`
	Index  hindex.Root  `json:"index"`
}

type Volume struct {
	Cell  cells.Cell
	Store cadata.Store
}

type Params struct {
	Volume Volume
}

type OID = hcorpus.Fingerprint

type Hoard struct {
	vol Volume

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

func (h *Hoard) Add(ctx context.Context, r io.Reader) (*OID, error) {
	var ret *OID
	store := h.vol.Store
	if err := h.update(ctx, func(s *State) (*State, error) {
		var croot *hcorpus.Root
		var iroot *hindex.Root
		// init
		if s != nil {
			croot = &s.Corpus
			iroot = &s.Index
		} else {
			var err error
			if croot, err = h.hcorpus.NewEmpty(ctx, store); err != nil {
				return nil, err
			}
			if iroot, err = h.hindex.NewEmpty(ctx, store); err != nil {
				return nil, err
			}
		}
		// add to corpus
		fp, croot2, err := h.hcorpus.Add(ctx, store, *croot, r)
		if err != nil {
			return nil, err
		}
		ret = &fp
		// add to index
		rs, err := h.hcorpus.Get(ctx, store, *croot2, *ret)
		if err != nil {
			return nil, err
		}
		tagSet := tagging.TagSet{}
		if err := taggers.SuggestTags(rs, tagSet); err != nil {
			return nil, err
		}
		iroot2, err := h.hindex.AddTags(ctx, store, *iroot, *ret, tagSet.Slice())
		if err != nil {
			return nil, err
		}
		return &State{
			Corpus: *croot2,
			Index:  *iroot2,
		}, nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (h *Hoard) Get(ctx context.Context, id OID) (io.ReadSeeker, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("no object with that id")
	}
	return h.hcorpus.Get(ctx, h.vol.Store, x.Corpus, id)
}

func (h *Hoard) GetTags(ctx context.Context, id OID) ([]tagging.Tag, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("no object with that id")
	}
	return h.hindex.GetTags(ctx, h.vol.Store, x.Index, id)
}

func (h *Hoard) ForEach(ctx context.Context, fn func(OID, []tagging.Tag) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	if x == nil {
		return nil
	}
	return h.hindex.ForEach(ctx, h.vol.Store, x.Index, fn)
}

func (h *Hoard) ListByPrefix(ctx context.Context, prefix []byte, limit int) (ret []OID, _ error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.hcorpus.ForEach(ctx, h.vol.Store, x.Corpus, prefix, func(fp OID) error {
		ret = append(ret, fp)
		if len(ret) >= limit {
			return tagging.ErrStopIter
		}
		return nil
	}); err != nil && err != tagging.ErrStopIter {
		return nil, err
	}
	return ret, nil
}

func (h *Hoard) ForEachTagKey(ctx context.Context, fn func(string) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	return h.hindex.ForEachTagKey(ctx, h.vol.Store, x.Index, fn)
}

func (h *Hoard) ForEachTagValue(ctx context.Context, tagKey string, fn func([]byte) error) error {
	x, err := h.get(ctx)
	if err != nil {
		return err
	}
	return h.hindex.ForEachTagValue(ctx, h.vol.Store, x.Index, tagKey, fn)
}

func (h *Hoard) Search(ctx context.Context, query tagging.Query) ([]OID, error) {
	x, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.hindex.Search(ctx, h.vol.Store, x.Index, query)
	if err != nil {
		return nil, err
	}
	return res.IDs, nil
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
