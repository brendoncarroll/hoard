package hoard

import (
	"encoding/json"
	"encoding/pem"

	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/webfs/pkg/webref"

	"github.com/brendoncarroll/blobcache/pkg/blobs"
)

const PEMType = "HOARD MANIFEST"

type Manifest struct {
	ID uint64 `json:"id"`

	WebRef     *webref.Ref `json:"webref"`
	PinSetName string      `json:"pinset_name"`
	PinSetRoot *blobs.ID   `json:"pinset_root"`
	BlobCount  uint64      `json:"blob_count"`

	Tags          taggers.TagSet `json:"tags"`
	SuggestedTags taggers.TagSet `json:"suggested_tags,omitempty"`
}

func (mf Manifest) Share() string {
	// Protocol buffers require marshaling like this
	refData, err := webref.Encode(webref.CodecJSON, mf.WebRef)
	if err != nil {
		panic(err)
	}
	x := struct {
		WebRef json.RawMessage `json:"webref"`
	}{refData}
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	b := &pem.Block{
		Type:  PEMType,
		Bytes: data,
	}
	return string(pem.EncodeToMemory(b))
}
