package hoardproto

import (
	"time"

	"github.com/brendoncarroll/hoard/pkg/tagdb"
	"github.com/brendoncarroll/hoard/pkg/taggers"
)

type TagSet = taggers.TagSet

type QueryReq struct {
	Query tagdb.Query `json:"query"`
	Limit int         `json:"limit"`

	Hops     int       `json:"hops"`
	Deadline time.Time `json:"deadline"`
}

type QueryRes struct {
	Manifests []*Manifest `json:"manifests"`
}
