package hoard

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/brendoncarroll/blobcache/pkg/blobcache"
	"github.com/brendoncarroll/blobcache/pkg/blobs"
	"github.com/brendoncarroll/webfs/pkg/webfsim"
	"github.com/brendoncarroll/webfs/pkg/webref"
)

const bcPrefix = "bc://"

type bcstore struct {
	pinSetName string
	bcn        *blobcache.Node
}

type rwStore struct {
	webref.Getter
	webref.Poster
}

func (s bcstore) Get(ctx context.Context, key string) ([]byte, error) {
	if !strings.HasPrefix(key, bcPrefix) {
		return nil, errors.New("must have blobcache prefix")
	}
	key = key[len(bcPrefix):]
	idBytes, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	id := blobs.ID{}
	copy(id[:], idBytes)
	return s.bcn.Get(ctx, id)
}

func (s bcstore) Post(ctx context.Context, prefix string, data []byte) (string, error) {
	if prefix != "" {
		return "", errors.New("prefix must be empty")
	}
	id, err := s.bcn.Post(ctx, s.pinSetName, data)
	if err != nil {
		return "", err
	}
	key := base64.RawURLEncoding.EncodeToString(id[:])
	return bcPrefix + key, nil
}

func (s bcstore) MaxBlobSize() int {
	return s.bcn.MaxBlobSize()
}

func makeStore(bcn *blobcache.Node, pinSetName string) webfsim.ReadPost {
	s1 := &webref.BasicStore{
		Store: bcstore{
			pinSetName: pinSetName,
			bcn:        bcn,
		},
	}
	return &rwStore{
		Poster: &webref.CryptoStore{
			EncAlgo:         webref.EncAlgo_CHACHA20,
			Inner:           s1,
			SecretSeed:      []byte{},
			ObfuscateLength: true,
		},
		Getter: s1,
	}
}
