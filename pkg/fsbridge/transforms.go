package fsbridge

import (
	"github.com/brendoncarroll/hoard/pkg/hoardfile"
)

type Transform struct {
	TransformInPlace func([]byte)
}

// HoardFileTransform encrypts blobs using the default
// convergent encryption strategy used by hoard
var HoardFileTransform = Transform{
	TransformInPlace: func(x []byte) {
		dek := hoardfile.DeriveKey(x)
		hoardfile.CryptoXOR(dek[:], x, x)
	},
}
