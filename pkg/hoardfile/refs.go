package hoardfile

import (
	"context"

	"github.com/blobcache/blobcache/pkg/bccrypto"
	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20"
)

const RefSize = blobs.IDSize + chacha20.KeySize

type Ref struct {
	ID  blobs.ID     `json:"id"`
	DEK bccrypto.DEK `json:"dek"`
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
	id, dek, err := bccrypto.Post(ctx, s, bccrypto.Convergent, ptext)
	if err != nil {
		return nil, err
	}
	return &Ref{
		ID:  id,
		DEK: *dek,
	}, nil
}

func getF(ctx context.Context, s blobs.Getter, ref Ref, fn func([]byte) error) error {
	return bccrypto.GetF(ctx, s, ref.DEK, ref.ID, fn)
}

func DeriveKey(ptext []byte) bccrypto.DEK {
	return bccrypto.Convergent(blobs.Hash(ptext))
}

func CryptoXOR(key, dst, src []byte) {
	nonce := [chacha20.NonceSize]byte{}
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce[:])
	if err != nil {
		panic(err)
	}
	cipher.XORKeyStream(dst, src)
}
