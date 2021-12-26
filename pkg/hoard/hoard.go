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
)

type Volume struct {
	Cell  cells.Cell
	Store cadata.Store
}

type Params struct {
	Corpus Volume
	Index  Volume
}

type OID = hcorpus.Fingerprint

type Hoard struct {
	corpus  Volume
	index   Volume
	indexes []Volume

	hindex  *hindex.Operator
	hcorpus *hcorpus.Operator
}

func New(params Params) *Hoard {
	return &Hoard{
		corpus:  params.Corpus,
		index:   params.Index,
		hindex:  hindex.New(),
		hcorpus: hcorpus.New(),
	}
}

func (h *Hoard) Add(ctx context.Context, r io.Reader) (*OID, error) {
	vol := h.corpus
	var ret *OID
	var root2 *hcorpus.Root
	if err := applyCorpus(ctx, vol.Cell, func(root *hcorpus.Root) (*hcorpus.Root, error) {
		if root == nil {
			var err error
			if root, err = h.hcorpus.NewEmpty(ctx, vol.Store); err != nil {
				return nil, err
			}
		}
		fp, root, err := h.hcorpus.Add(ctx, vol.Store, *root, r)
		if err != nil {
			return nil, err
		}
		ret = &fp
		root2 = root
		return root, nil
	}); err != nil {
		return nil, nil
	}
	rs, err := h.hcorpus.Get(ctx, vol.Store, *root2, *ret)
	if err != nil {
		return nil, err
	}
	tagSet := tagging.TagSet{}
	if err := taggers.SuggestTags(rs, tagSet); err != nil {
		return nil, err
	}
	vol = h.index
	if err := applyIndex(ctx, vol.Cell, func(root *hindex.Root) (*hindex.Root, error) {
		if root == nil {
			var err error
			if root, err = h.hindex.NewEmpty(ctx, vol.Store); err != nil {
				return nil, err
			}
		}
		return h.hindex.AddTags(ctx, vol.Store, *root, *ret, tagSet.Slice())
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (h *Hoard) Get(ctx context.Context, id OID) (io.ReadSeeker, error) {
	vol := h.corpus
	root, err := getCorpus(ctx, vol.Cell)
	if err != nil {
		return nil, err
	}
	return h.hcorpus.Get(ctx, vol.Store, *root, id)
}

func (h *Hoard) ListByPrefix(ctx context.Context, prefix []byte, limit int) (ret []OID, _ error) {
	vol := h.corpus
	root, err := getCorpus(ctx, vol.Cell)
	if err != nil {
		return nil, err
	}
	if err := h.hcorpus.ForEach(ctx, vol.Store, *root, prefix, func(fp OID) error {
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

func (h *Hoard) ListTags(ctx context.Context, fn func(id OID, tag tagging.Tag) error) error {
	vol := h.index
	root, err := getIndex(ctx, vol.Cell)
	if err != nil {
		return err
	}
	qb := h.hindex.NewQueryBackend(vol.Store, *root)
	return qb.Scan(ctx, tagging.Span{}, func(id OID, key, value []byte) error {
		return fn(id, tagging.Tag{Key: string(key), Value: string(value)})
	})
}

func (h *Hoard) Search(ctx context.Context, query tagging.Query) ([]OID, error) {
	vol := h.index
	root, err := getIndex(ctx, vol.Cell)
	if err != nil {
		return nil, err
	}
	res, err := h.hindex.Search(ctx, vol.Store, *root, query)
	if err != nil {
		return nil, err
	}
	return res.IDs, nil
}

func getIndex(ctx context.Context, c cells.Cell) (*hindex.Root, error) {
	var x hindex.Root
	if err := getJSON(ctx, c, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func getCorpus(ctx context.Context, c cells.Cell) (*hcorpus.Root, error) {
	var x hcorpus.Root
	if err := getJSON(ctx, c, &x); err != nil {
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

func applyCorpus(ctx context.Context, cell cells.Cell, fn func(root *hcorpus.Root) (*hcorpus.Root, error)) error {
	return cells.Apply(ctx, cell, func(data []byte) ([]byte, error) {
		var x *hcorpus.Root
		if len(data) > 0 {
			x = &hcorpus.Root{}
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

func applyIndex(ctx context.Context, cell cells.Cell, fn func(root *hindex.Root) (*hindex.Root, error)) error {
	return cells.Apply(ctx, cell, func(data []byte) ([]byte, error) {
		var x *hindex.Root
		if len(data) > 0 {
			x = &hindex.Root{}
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
