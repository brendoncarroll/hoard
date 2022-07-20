package labels

import (
	"fmt"
	"time"
)

type Pair struct {
	Key   string
	Value []byte
}

func (t Pair) VInt64() (int64, error) {
	panic("")
}

func (t Pair) VTime() (time.Time, error) {
	panic("")
}

func (t Pair) String() string {
	return fmt.Sprintf("(%s=>%q)", t.Key, t.Value)
}

type PairSet map[string][]byte

func (ts PairSet) Slice() (ret []Pair) {
	for k, v := range ts {
		ret = append(ret, Pair{k, v})
	}
	return ret
}
