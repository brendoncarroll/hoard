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
	span2 := prefixSpan(gotkv.Span{Start: span.Begin, End: span.End}, []byte{'f', 0x00})
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

func (qb QueryBackend) ScanInverted(ctx context.Context, tagKey string, fn tagging.IterFunc) error {
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
	start := prefix
	start = append(start, x.Start...)
	end := gotkv.PrefixEnd(prefix)
	if x.End != nil {
		end = prefix
		end = append(end, x.End...)
	}
	return gotkv.Span{
		Start: start,
		End:   end,
	}
}
