package tagging

import (
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/brendoncarroll/go-state"
	"github.com/brendoncarroll/hoard/pkg/hcorpus"
	"github.com/gotvc/got/pkg/gotkv"
)

type ID = hcorpus.Fingerprint

type ResultSet struct {
	IDs                  []ID
	Offset, Count, Total int
}

type PredicateOp string

const (
	OpNone = PredicateOp("NONE")
	OpAny  = PredicateOp("ANY")

	OpEq = PredicateOp("=")
	OpLt = PredicateOp("<")
	OpGt = PredicateOp(">")

	OpContains = PredicateOp("CONTAINS")
	OpRegexp   = PredicateOp("REGEXP")

	OpIn = PredicateOp("IN")

	OpOR  = PredicateOp("OR")
	OpAND = PredicateOp("AND")
)

type Query struct {
	Where Predicate `json:"where"`
	Limit int       `json:"limit"`
}

type Predicate struct {
	Op PredicateOp `json:"op"`

	Key string `json:"key"`

	Value      string   `json:"value,omitempty"`
	Values     []string `json:"values,omitempty"`
	SubQueries []Query  `json:"sub_queries,omitempty"`

	Limit int `json:"limit"`
}

var ErrStopIter = fmt.Errorf("stop iteration")

type IterFunc = func(id ID, key, value []byte) error

type Span = state.ByteRange

type QueryBackend interface {
	Scan(ctx context.Context, keySpan Span, fn IterFunc) error
	ScanInverted(ctx context.Context, keySpan Span, fn IterFunc) error
	GetValue(ctx context.Context, id ID, key []byte) ([]byte, error)
}

func DoQuery(ctx context.Context, be QueryBackend, q Query) (*ResultSet, error) {
	if q.Where.Op == PredicateOp("") {
		q.Where.Op = OpAny
	}

	ids := map[ID]int{}
	if err := query(ctx, be, ids, q, false); err != nil {
		return nil, err
	}

	resultSet := &ResultSet{
		IDs:    make([]ID, 0, len(ids)),
		Count:  len(ids),
		Offset: 0,
		Total:  -1,
	}
	for id := range ids {
		resultSet.IDs = append(resultSet.IDs, id)
	}
	return resultSet, nil
}

func query(ctx context.Context, be QueryBackend, ids map[ID]int, q Query, pruning bool) error {
	switch q.Where.Op {
	case OpOR:
		ids2 := map[ID]int{}
		if err := queryOR(ctx, be, ids2, q.Limit, q.Where.SubQueries); err != nil {
			return err
		}
		count := 0
		for id := range ids2 {
			if count >= q.Limit {
				break
			}
			ids[id]++
			count++
		}
	case OpAND:
		ids2 := map[ID]int{}
		if err := queryAND(ctx, be, ids, q.Where.SubQueries); err != nil {
			return err
		}
		count := 0
		for id := range ids2 {
			if count >= q.Limit {
				break
			}
			ids[id]++
			count++
		}
	case OpAny:
		err := be.Scan(ctx, Span{}, func(id ID, _, value []byte) error {
			ids[id]++
			if len(ids) > q.Limit {
				return ErrStopIter
			}
			return nil
		})
		if err == ErrStopIter {
			err = nil
		}
		return err

	default:
		if pruning {
			return scanResults(ctx, be, ids, q.Where, func(id ID) bool {
				ids[id]++
				return len(ids) < 1
			})
		} else {
			return scanTable(ctx, be, q.Where, func(id ID) bool {
				ids[id]++
				return len(ids) < q.Limit
			})
		}
	}
	return nil
}

func queryAND(ctx context.Context, be QueryBackend, ids map[ID]int, subs []Query) error {
	round := 0
	for _, q := range subs {
		if err := query(ctx, be, ids, q, round == 0); err != nil {
			return err
		}
		round++
		for id, count := range ids {
			if count < round {
				delete(ids, id)
			}
		}
	}
	return nil
}

func queryOR(ctx context.Context, be QueryBackend, ids map[ID]int, limit int, subs []Query) error {
	for _, q := range subs {
		if err := query(ctx, be, ids, q, false); err != nil {
			return err
		}
		if len(ids) >= limit {
			break
		}
	}
	return nil
}

func scanResults(ctx context.Context, be QueryBackend, ids map[ID]int, pred Predicate, fn func(id ID) bool) error {
	predFunc, err := makePredicateFunc(pred)
	if err != nil {
		return err
	}
	for id := range ids {
		value, err := be.GetValue(ctx, id, []byte(pred.Key))
		if err != nil {
			return err
		}
		if predFunc(value) {
			if !fn(id) {
				break
			}
		}
	}
	return nil
}

func scanTable(ctx context.Context, be QueryBackend, pred Predicate, fn func(id ID) bool) error {
	switch pred.Op {
	case OpEq, OpLt, OpGt, OpContains, OpIn, OpAny:
		predFunc, err := makePredicateFunc(pred)
		if err != nil {
			return err
		}
		span := Span{Begin: []byte(pred.Key), End: gotkv.KeyAfter([]byte(pred.Key))}
		err = be.ScanInverted(ctx, span, func(id ID, _, value []byte) error {
			if predFunc(value) {
				if !fn(id) {
					return ErrStopIter
				}
			}
			return nil
		})
		if err == ErrStopIter {
			err = nil
		}
		return err
	case OpNone:
		return nil
	default:
		return errInvalidOp(pred.Op)
	}
}

func makePredicateFunc(pred Predicate) (func([]byte) bool, error) {
	target := []byte(pred.Value)
	var fn func([]byte) bool
	switch pred.Op {
	case OpEq:
		fn = func(value []byte) bool {
			return bytes.Equal(value, target)
		}
	case OpLt:
		fn = func(value []byte) bool {
			return bytes.Compare(value, target) < 0
		}
	case OpGt:
		fn = func(value []byte) bool {
			return bytes.Compare(value, target) > 0
		}
	case OpContains:
		fn = func(value []byte) bool {
			return bytes.Contains(value, target)
		}
	case OpIn:
		fn = func(value []byte) bool {
			for _, pv := range pred.Values {
				if bytes.Equal(value, []byte(pv)) {
					return true
				}
			}
			return false
		}
	case OpAny:
		fn = func([]byte) bool { return true }
	case OpNone:
		fn = func([]byte) bool { return false }
	case OpRegexp:
		re, err := regexp.Compile(pred.Value)
		if err != nil {
			return nil, err
		}
		fn = re.Match
	default:
		return nil, errInvalidOp(pred.Op)
	}
	return fn, nil
}
