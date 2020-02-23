package tagdb

import (
	"bytes"
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

type ResultSet struct {
	IDs                  []uint64
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

	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`

	SubQueries []Query `json:"value"`

	Limit int `json:"limit"`
}

func (tdb *TagDB) Query(ctx context.Context, q Query) (*ResultSet, error) {
	log.Infof("tagdb.Query %+v", q)
	if q.Where.Op == PredicateOp("") {
		q.Where.Op = OpAny
	}

	ids := map[uint64]int{}
	if err := tdb.query(ctx, ids, q, false); err != nil {
		return nil, err
	}

	resultSet := &ResultSet{
		IDs:    make([]uint64, 0, len(ids)),
		Count:  len(ids),
		Offset: 0,
		Total:  -1,
	}
	for id := range ids {
		resultSet.IDs = append(resultSet.IDs, id)
	}
	return resultSet, nil
}

func (tdb *TagDB) query(ctx context.Context, ids map[uint64]int, q Query, pruning bool) error {
	switch q.Where.Op {
	case OpOR:
		ids2 := map[uint64]int{}
		if err := tdb.queryOR(ctx, ids2, q.Limit, q.Where.SubQueries); err != nil {
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
		ids2 := map[uint64]int{}
		if err := tdb.queryAND(ctx, ids, q.Where.SubQueries); err != nil {
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
		return tdb.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucketTags))

			tagc := b.Cursor()
			for k, _ := tagc.First(); k != nil; k, _ = tagc.Next() {
				tagb := b.Bucket(k)
				forward := tagb.Bucket([]byte("f"))

				c := forward.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					id := bytesToID(k)
					ids[id]++
					if len(ids) >= q.Limit {
						return nil
					}
				}
			}
			return nil
		})

	default:
		if pruning {
			return tdb.scanResults(ctx, ids, q.Where, func(id uint64) bool {
				ids[id]++
				return len(ids) < 1
			})
		} else {
			return tdb.scanTable(ctx, q.Where, func(id uint64) bool {
				ids[id]++
				return len(ids) < q.Limit
			})
		}
	}
	return nil
}

func (tdb *TagDB) queryAND(ctx context.Context, ids map[uint64]int, subs []Query) error {
	round := 0
	for _, q := range subs {
		if err := tdb.query(ctx, ids, q, round == 0); err != nil {
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

func (tdb *TagDB) queryOR(ctx context.Context, ids map[uint64]int, limit int, subs []Query) error {
	for _, q := range subs {
		if err := tdb.query(ctx, ids, q, false); err != nil {
			return err
		}
		if len(ids) >= limit {
			break
		}
	}
	return nil
}

func (tdb *TagDB) scanResults(ctx context.Context, ids map[uint64]int, pred Predicate, fn func(id uint64) bool) error {
	for id := range ids {
		value, err := tdb.GetTag(ctx, id, pred.Key)
		if err != nil {
			return err
		}
		x := pred.Value
		switch pred.Op {
		case OpEq:
			if strings.Compare(value, x) == 0 {
				fn(id)
			}
		case OpLt:
			if strings.Compare(value, x) < 0 {
				fn(id)
			}
		case OpGt:
			if strings.Compare(value, x) > 0 {
				fn(id)
			}
		case OpContains:
			if strings.Contains(value, x) {
				fn(id)
			}
		case OpIn:
			for _, pv := range pred.Values {
				if strings.Compare(value, pv) == 0 {
					fn(id)
					break
				}
			}
		case OpAny:
			fn(id)
		case OpNone:
			return nil
		default:
			return errInvalidOp(pred.Op)
		}
	}
	return nil
}

func (tdb *TagDB) scanTable(ctx context.Context, pred Predicate, f func(id uint64) bool) error {
	err := tdb.db.View(func(tx *bolt.Tx) error {
		_, inv, err := bucketsForTag(tx, pred.Key)
		if err == bolt.ErrBucketNotFound {
			return nil
		}
		if err != nil {
			return err
		}

		x := []byte(pred.Value)
		fn := func(id uint64, v []byte) bool {
			return f(id)
		}

		switch pred.Op {
		case OpEq:
			err = forEachEq(inv, x, fn)
		case OpLt:
			err = forEachLt(inv, x, fn)
		case OpGt:
			err = forEachGt(inv, x, fn)
		case OpContains:
			err = forEachContains(inv, x, fn)
		case OpIn:
			err = forEachIn(inv, pred.Values, fn)
		case OpAny:
			err = forEachGt(inv, []byte{}, fn)
		case OpNone:
			return nil
		default:
			return errInvalidOp(pred.Op)
		}
		return err
	})
	return err
}

func forEachEq(inv *bolt.Bucket, x []byte, fn func(id uint64, value []byte) bool) error {
	c := inv.Cursor()
	for k, _ := c.Seek([]byte(x)); k != nil; k, _ = c.Next() {
		id, value, err := splitInvKey(k)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(value, []byte(x)) {
			break
		}
		if !fn(id, value) {
			break
		}
	}
	return nil
}

func forEachLt(inv *bolt.Bucket, x []byte, fn func(id uint64, value []byte) bool) error {
	c := inv.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		id, value, err := splitInvKey(k)
		if err != nil {
			return err
		}
		if bytes.Compare(value, []byte(x)) >= 0 {
			break
		}
		if !fn(id, value) {
			break
		}
	}
	return nil
}

func forEachGt(inv *bolt.Bucket, x []byte, fn func(id uint64, value []byte) bool) error {
	c := inv.Cursor()
	for k, _ := c.Seek(x); k != nil; k, _ = c.Next() {
		id, value, err := splitInvKey(k)
		if err != nil {
			return err
		}
		if bytes.Compare(value, []byte(x)) < 0 {
			continue
		}
		if !fn(id, value) {
			break
		}
	}
	return nil
}

func forEachContains(inv *bolt.Bucket, x []byte, fn func(id uint64, value []byte) bool) error {
	c := inv.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		id, value, err := splitInvKey(k)
		if err != nil {
			return err
		}
		if bytes.Contains(value, []byte(x)) {
			if !fn(id, value) {
				break
			}
		}
	}
	return nil
}

func forEachIn(inv *bolt.Bucket, xs []string, fn func(id uint64, value []byte) bool) error {
	for _, x := range xs {
		err := forEachEq(inv, []byte(x), fn)
		if err != nil {
			return err
		}
	}
	return nil
}
