package tagging

import (
	"fmt"
	"time"
)

type Tag struct {
	Key   string
	Value []byte
}

func (t Tag) VInt64() (int64, error) {
	panic("")
}

func (t Tag) VTime() (time.Time, error) {
	panic("")
}

func (t Tag) String() string {
	return fmt.Sprintf("%s=>%q", t.Key, t.Value)
}

type TagSet map[string][]byte

func (ts TagSet) Slice() (ret []Tag) {
	for k, v := range ts {
		ret = append(ret, Tag{k, v})
	}
	return ret
}
