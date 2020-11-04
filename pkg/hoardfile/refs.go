package hoardfile

import (
	"context"
	"encoding/base64"

	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20"
)

const RefSize = blobs.IDSize + chacha20.KeySize

type DEK [32]byte

func (dek DEK) MarshalJSON() ([]byte, error) {
	return []byte(`"` + base64.RawURLEncoding.EncodeToString(dek[:]) + `"`), nil
}

func (dek *DEK) UnmarshalJSON(data []byte) error {
	_, err := base64.RawURLEncoding.Decode(dek[:], data[1:len(data)-1])
	return err
}

type Ref struct {
	ID  blobs.ID `json:"id"`
	DEK DEK      `json:"dek"`
}

func RefFromBytes(x []byte) (*Ref, error) {
	if len(x) != RefSize {
		return nil, errors.Errorf("incorrect length for ref")
	}
	ref := &Ref{}
	copy(ref.ID[:], x[:blobs.IDSize])
	copy(ref.DEK[:], x[blobs.IDSize:])
	return ref, nil
}

func post(ctx context.Context, s blobs.Poster, ptext []byte) (*Ref, error) {
	ctext := make([]byte, len(ptext))
	dek := blobs.Hash(ptext)
	CryptoXOR(dek[:], ctext, ptext)
	id, err := s.Post(ctx, ctext)
	if err != nil {
		return nil, err
	}
	return &Ref{ID: id, DEK: DEK(dek)}, nil
}

func getF(ctx context.Context, s blobs.Getter, ref Ref, fn func([]byte) error) error {
	return s.GetF(ctx, ref.ID, func(ctext []byte) error {
		ptext := make([]byte, len(ctext))
		CryptoXOR(ref.DEK[:], ptext, ctext)
		return fn(ptext)
	})
}

func DeriveKey(ptext []byte) [32]byte {
	return blobs.Hash(ptext)
}

func CryptoXOR(key, dst, src []byte) {
	nonce := [chacha20.NonceSize]byte{}
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce[:])
	if err != nil {
		panic(err)
	}
	cipher.XORKeyStream(dst, src)
}
