package hindex

import (
	"context"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/hoard/pkg/labels"
	"github.com/gotvc/got/pkg/gotkv"
)

var _ labels.QueryBackend = QueryBackend{}

type QueryBackend struct {
	op   *Operator
	s    cadata.Store
	root Root
}

func (qb QueryBackend) ScanForward(ctx context.Context, span labels.Span, fn labels.IterFunc) error {
	span2 := prefixSpan(gotkv.Span{Begin: span.Begin, End: span.End}, []byte{'f', 0x00})
	return qb.op.gotkv.ForEach(ctx, qb.s, qb.root, span2, func(ent gotkv.Entry) error {
		fp, key, value, err := parseForwardEntry(ent)
		if err != nil {
			return err
		}
		return fn(fp, key, value)
	})
}

func (qb QueryBackend) GetValue(ctx context.Context, id OID, key string) ([]byte, error) {
	return qb.op.gotkv.Get(ctx, qb.s, qb.root, makeForwardKey(nil, id, []byte(key)))
}

func (qb QueryBackend) ScanInverted(ctx context.Context, tagKey string, fn labels.IterFunc) error {
	var span gotkv.Span
	if tagKey != "" {
		span = prefixSpan(gotkv.PrefixSpan([]byte(tagKey)), []byte{'i', 0x00})
	}
	return qb.op.gotkv.ForEach(ctx, qb.s, qb.root, span, func(ent gotkv.Entry) error {
		fp, key, value, err := parseInverseEntry(ent)
		if err != nil {
			return err
		}
		return fn(*fp, key, value)
	})
}

func prefixSpan(x gotkv.Span, prefix []byte) gotkv.Span {
	begin := prefix
	begin = append(begin, x.Begin...)
	end := gotkv.PrefixEnd(prefix)
	if x.End != nil {
		end = prefix
		end = append(end, x.End...)
	}
	return gotkv.Span{
		Begin: begin,
		End:   end,
	}
}
