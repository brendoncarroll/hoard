package hoard

import (
	"github.com/brendoncarroll/blobcache/pkg/blobs"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
)

type Manifest struct {
	hoardproto.Manifest

	// blobcache
	PinSetName string    `json:"pinset_name"`
	PinSetRoot *blobs.ID `json:"pinset_root"`
}

type ResultSet struct {
	Manifests []*Manifest `json:"manifests"`

	Offest int `json:"offset"`
	Count  int `json:"limit"`
	Total  int `json:"total"`
}
