package hoard

import (
	"github.com/blobcache/blobcache/pkg/blobcache"
	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
)

type Manifest struct {
	hoardproto.Manifest

	// blobcache
	PinSetID   blobcache.PinSetID `json:"pinset_id"`
	PinSetRoot *blobs.ID          `json:"pinset_root"`
}

type ResultSet struct {
	Manifests []*Manifest `json:"manifests"`

	Offest int `json:"offset"`
	Count  int `json:"limit"`
	Total  int `json:"total"`
}
