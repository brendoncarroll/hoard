package hindex

import (
	"context"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/hoard/pkg/tagging"
	"github.com/gotvc/got/pkg/gotkv"
)

var _ tagging.QueryBackend = QueryBackend{}

type QueryBackend struct {
	op   *Operator
	s    cadata.Store
	root Root
}

func (qb QueryBackend) Scan(ctx context.Context, span tagging.Span, fn tagging.IterFunc) error {
	span2 := transformSpan(span, []byte{'f'})
	return qb.op.gotkv.ForEach(ctx, qb.s, qb.root, span2, func(ent gotkv.Entry) error {
		key, fp, err := parseForwardKey(ent.Key)
		if err != nil {
			return err
		}
		return fn(fp, key, ent.Value)
	})
}

func (qb QueryBackend) GetValue(ctx context.Context, id Fingerprint, key []byte) ([]byte, error) {
	return qb.op.gotkv.Get(ctx, qb.s, qb.root, makeForwardKey(nil, key, id))
}

func (qb QueryBackend) ScanInverted(ctx context.Context, span tagging.Span, fn tagging.IterFunc) error {
	span2 := transformSpan(span, []byte{'i'})
	return qb.op.gotkv.ForEach(ctx, qb.s, qb.root, span2, func(ent gotkv.Entry) error {
		key, value, fp, err := parseInverseKey(ent.Key)
		if err != nil {
			return err
		}
		return fn(*fp, key, value)
	})
}

func transformSpan(x tagging.Span, prefix []byte) gotkv.Span {
	start := append(prefix, 0x00)
	start = append(start, x.Begin...)

	end := gotkv.PrefixEnd(append(prefix, 0x00))
	if x.End != nil {
		end = append(prefix, 0x00)
		end = append(end, x.End...)
	}

	return gotkv.Span{
		Start: start,
		End:   end,
	}
}
