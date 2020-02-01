package hoardproto

import (
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/brendoncarroll/blobcache/pkg/blobs"
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/webfs/pkg/webref"
)

const PEMType = "HOARD MANIFEST"

type Manifest struct {
	Peer p2p.PeerID `json:"peer,omitempty"`
	ID   uint64     `json:"id"`

	WebRef     *webref.Ref `json:"webref"`
	PinSetName string      `json:"pinset_name"`
	PinSetRoot *blobs.ID   `json:"pinset_root"`
	BlobCount  uint64      `json:"blob_count"`

	Tags          taggers.TagSet `json:"tags"`
	SuggestedTags taggers.TagSet `json:"suggested_tags,omitempty"`
}

func (mf Manifest) String() string {
	return mf.Peer.String() + "/" + fmt.Sprint(mf.ID)
}

func (mf Manifest) Share() string {
	x := struct {
		WebRef *webref.Ref `json:"webref"`
	}{mf.WebRef}
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
