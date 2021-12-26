package tagging

import "time"

type Tag struct {
	Key, Value string
}

func (t Tag) VInt64() (int64, error) {
	panic("")
}

func (t Tag) VTime() (time.Time, error) {
	panic("")
}

type TagSet map[string]string

func (ts TagSet) Slice() (ret []Tag) {
	for k, v := range ts {
		ret = append(ret, Tag{k, v})
	}
	return ret
}
