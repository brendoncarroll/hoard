package hoardfile

import (
	"context"
	"io"

	"github.com/blobcache/blobcache/pkg/blobs"
)

var _ io.ReadSeeker = &Reader{}

type Reader struct {
	ctx    context.Context
	store  blobs.Getter
	file   File
	offset int64
}

func NewReader(ctx context.Context, s blobs.Getter, file File) *Reader {
	return &Reader{
		ctx:   ctx,
		store: s,
		file:  file,
	}
}

func (r *Reader) Read(data []byte) (int, error) {
	n, err := ReadAt(r.ctx, r.store, r.file, int(r.offset), data)
	r.offset += int64(n)
	return n, err
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = int64(r.file.Size) - offset
	default:
		panic("invalid whence")
	}
	return r.offset, nil
}
