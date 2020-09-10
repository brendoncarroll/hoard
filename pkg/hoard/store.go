package hoard

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/blobcache/blobcache/pkg/blobcache"
	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/brendoncarroll/webfs/pkg/webfsim"
	"github.com/brendoncarroll/webfs/pkg/webref"
)

const bcPrefix = "bc://"

type bcstore struct {
	pinset blobcache.PinSetID
	bcn    *blobcache.Node
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
	id := blobs.ID{}
	if err := id.UnmarshalB64([]byte(key)); err != nil {
		return nil, err
	}
	var data []byte
	if err := s.bcn.GetF(ctx, id, func(x []byte) error {
		data = append([]byte{}, x...)
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}

func (s bcstore) Post(ctx context.Context, prefix string, data []byte) (string, error) {
	if prefix != "" {
		return "", errors.New("prefix must be empty")
	}
	id, err := s.bcn.Post(ctx, s.pinset, data)
	if err != nil {
		return "", err
	}
	key := base64.RawURLEncoding.EncodeToString(id[:])
	return bcPrefix + key, nil
}

func (s bcstore) MaxBlobSize() int {
	return s.bcn.MaxBlobSize()
}

func makeStore(bcn *blobcache.Node, pinset blobcache.PinSetID) webfsim.ReadPost {
	s1 := &webref.BasicStore{
		Store: bcstore{
			pinset: pinset,
			bcn:    bcn,
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
