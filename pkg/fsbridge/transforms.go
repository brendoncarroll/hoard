package fsbridge

import (
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/sha3"
)

type Transform struct {
	TransformInPlace func([]byte)
}

var WebFSTransform = Transform{
	// WebFSTransform encrypts blobs using the default
	// convergent encryption strategy used by WebFS
	TransformInPlace: func(x []byte) {
		// derive key
		id := sha3.Sum256(x)
		dek := sha3.Sum256(id[:])
		nonce := [chacha20.NonceSize]byte{} // 0s
		ciph, err := chacha20.NewUnauthenticatedCipher(dek[:], nonce[:])
		if err != nil {
			panic(err)
		}
		ctext := x // inplace
		ciph.XORKeyStream(ctext, x)
	},
}
