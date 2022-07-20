package hcorpus

import (
	"context"
	"errors"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/gotvc/got/pkg/gotkv"
)

const MaxDataSize = 4096

type Root gotfs.Root

type Operator struct {
	gotkv gotkv.Operator
}

func New() *Operator {
	return &Operator{
		gotkv: gotkv.NewOperator(1<<13, 1<<20),
	}
}

func (o *Operator) NewEmpty(ctx context.Context, s cadata.Store) (*Root, error) {
	r, err := o.gotkv.NewEmpty(ctx, s)
	if err != nil {
		return nil, err
	}
	return (*Root)(r), nil
}

func (o *Operator) Post(ctx context.Context, s cadata.Store, x Root, data []byte) (ID, *Root, error) {
	if len(data) > MaxDataSize {
		return ID{}, nil, errors.New("value too large")
	}
	id := Hash(data)
	root, err := o.gotkv.Put(ctx, s, gotkv.Root(x), id[:], data)
	if err != nil {
		return ID{}, nil, err
	}
	return id, (*Root)(root), nil
}

func (o *Operator) Get(ctx context.Context, s cadata.Store, x Root, fp ID) ([]byte, error) {
	return o.gotkv.Get(ctx, s, gotkv.Root(x), fp[:])
}

func (o *Operator) ForEach(ctx context.Context, s cadata.Store, x Root, span gotkv.Span, fn func(fp ID) error) error {
	return o.gotkv.ForEach(ctx, s, gotkv.Root(x), span, func(ent gotkv.Entry) error {
		id := IDFromBytes(ent.Key)
		return fn(id)
	})
}

func (o *Operator) Delete(ctx context.Context, s cadata.Store, x Root, id ID) (*Root, error) {
	y, err := o.gotkv.Delete(ctx, s, gotkv.Root(x), id[:])
	return (*Root)(y), err
}
