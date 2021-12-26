package hcorpus

import (
	"bytes"
	"context"
	"encoding/hex"
	"hash"
	"io"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

type Fingerprint [32]byte

func (fp Fingerprint) String() string {
	return fp.HexString()
}

func (fp Fingerprint) HexString() string {
	return hex.EncodeToString(fp[:])
}

func FPFromBytes(x []byte) (ret Fingerprint) {
	copy(ret[:], x)
	return ret
}

type Fingerprinter struct {
	h hash.Hash
}

func NewFingerprinter() Fingerprinter {
	h, _ := blake2b.New(32, nil)
	return Fingerprinter{
		h: h,
	}
}

func (f Fingerprinter) Write(data []byte) (int, error) {
	return f.h.Write(data)
}

func (f Fingerprinter) Finish() (ret Fingerprint) {
	f.h.Sum(ret[:0])
	return ret
}

type Root gotfs.Root

type Operator struct {
	gotfs gotfs.Operator
}

func New() *Operator {
	return &Operator{
		gotfs: gotfs.NewOperator(),
	}
}

func (o *Operator) NewEmpty(ctx context.Context, s cadata.Store) (*Root, error) {
	r, err := o.gotfs.NewEmpty(ctx, s)
	if err != nil {
		return nil, err
	}
	return (*Root)(r), nil
}

func (o *Operator) Add(ctx context.Context, s cadata.Store, x Root, r io.Reader) (Fingerprint, *Root, error) {
	h := NewFingerprinter()
	r2 := io.TeeReader(r, h)
	fileRoot, err := o.gotfs.CreateFileRoot(ctx, s, s, r2)
	if err != nil {
		return Fingerprint{}, nil, err
	}
	fp := h.Finish()
	root, err := o.gotfs.Graft(ctx, s, s, gotfs.Root(x), fp.HexString(), *fileRoot)
	if err != nil {
		return Fingerprint{}, nil, err
	}
	return fp, (*Root)(root), nil
}

func (o *Operator) Get(ctx context.Context, s cadata.Store, x Root, fp Fingerprint) (io.ReadSeeker, error) {
	if _, err := o.gotfs.GetFileMetadata(ctx, s, gotfs.Root(x), fp.HexString()); err != nil {
		return nil, err
	}
	return o.gotfs.NewReader(ctx, s, s, gotfs.Root(x), fp.HexString()), nil
}

func (o *Operator) ForEach(ctx context.Context, s cadata.Store, x Root, prefix []byte, fn func(fp Fingerprint) error) error {
	return o.gotfs.ForEachFile(ctx, s, gotfs.Root(x), "", func(p string, _ *gotfs.Metadata) error {
		var fp Fingerprint
		if n, err := hex.Decode(fp[:], []byte(p)); err != nil {
			return err
		} else if n < len(fp) {
			return errors.Errorf("short path %s", p)
		}
		if bytes.HasPrefix(fp[:], prefix) {
			return fn(fp)
		}
		return nil
	})
}

func (o *Operator) Delete(ctx context.Context, x Root, fp [32]byte) (*Root, error) {
	panic("not implemented")
}
