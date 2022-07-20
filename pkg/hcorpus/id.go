package hcorpus

import (
	"encoding/hex"

	"lukechampine.com/blake3"
)

type ID [32]byte

func (id ID) String() string {
	return id.HexString()
}

func (id ID) HexString() string {
	return hex.EncodeToString(id[:])
}

func IDFromBytes(x []byte) (ret ID) {
	copy(ret[:], x)
	return ret
}

func Hash(data []byte) ID {
	return blake3.Sum256(data)
}
