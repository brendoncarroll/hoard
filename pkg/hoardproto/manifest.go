package hoardproto

import (
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/webfs/pkg/webref"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

type Fingerprint = blobs.ID

// FQID is a fully quallified ID
type FQID struct {
	Peer p2p.PeerID
	ID   uint64
}

func (fqid FQID) String() string {
	return fmt.Sprintf("%v/%d", fqid.Peer, fqid.ID)
}

type Manifest struct {
	Peer      p2p.PeerID     `json:"peer,omitempty"`
	ID        uint64         `json:"id"`
	WebRef    *webref.Ref    `json:"webref"`
	BlobCount uint64         `json:"blob_count"`
	Tags      taggers.TagSet `json:"tags"`
}

func (mf Manifest) String() string {
	return fmt.Sprintf("Manifest{%v}", mf.FQID())
}

func (mf Manifest) FQID() FQID {
	return FQID{
		Peer: mf.Peer,
		ID:   mf.ID,
	}
}

func (mf Manifest) Fingerprint() Fingerprint {
	data, _ := proto.Marshal(mf.WebRef)
	return blobs.Hash(data)
}

const ManifestPEMType = "HOARD MANIFEST"

func (mf Manifest) Share() string {
	x := struct {
		WebRef *webref.Ref `json:"webref"`
	}{mf.WebRef}
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	b := &pem.Block{
		Type:  ManifestPEMType,
		Bytes: data,
	}
	return string(pem.EncodeToMemory(b))
}

func ManifestFromPEM(data []byte) (*Manifest, error) {
	block, _ := pem.Decode(data)
	if block.Type != ManifestPEMType {
		return nil, errors.Errorf("wrong PEM type %v", block.Type)
	}
	mf := &Manifest{}
	if err := json.Unmarshal(block.Bytes, &mf); err != nil {
		return nil, err
	}
	return mf, nil
}
