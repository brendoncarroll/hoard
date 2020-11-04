package hoardfile

import (
	"context"
	"io"
	"math/bits"

	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/pkg/errors"
)

type File struct {
	Size uint64 `json:"size"`
	Root Ref    `json:"root"`
}

func ReadAt(ctx context.Context, s blobs.Getter, x File, offset int, buf []byte) (n int, err error) {
	level := depth(x.Size)
	for i := offset / blobs.MaxSize; n < len(buf) && offset < int(x.Size); i = offset / blobs.MaxSize {
		ref, err := getPiece(ctx, s, x.Root, level, i)
		if err != nil {
			return n, err
		}
		if err := getF(ctx, s, *ref, func(data []byte) error {
			relOffset := offset % blobs.MaxSize
			n2 := copy(buf[n:], data[relOffset:])
			n += n2
			offset += n2
			return nil
		}); err != nil {
			return n, err
		}
	}
	if uint64(offset) == x.Size {
		return n, io.EOF
	}
	return n, nil
}

func getPiece(ctx context.Context, s blobs.Getter, root Ref, level, i int) (*Ref, error) {
	if i < 0 {
		panic(i)
	}
	if level == 0 {
		return &root, nil
	}
	var ref Ref
	if err := getF(ctx, s, root, func(data []byte) error {
		idx, err := newIndexUsing(data)
		if err != nil {
			return err
		}
		ref = idx.Get(i * BranchingFactor / exp(level))
		return nil
	}); err != nil {
		return nil, err
	}
	return getPiece(ctx, s, ref, level-1, i%BranchingFactor)
}

func Create(ctx context.Context, s blobs.Poster, r io.Reader) (*File, error) {
	buf := make([]byte, blobs.MaxSize)

	indexes := []Index{newIndex()}
	counts := []int{0}
	var size int
	for done := false; !done; {
		n, err := io.ReadFull(r, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		size += n
		done = err == io.EOF || err == io.ErrUnexpectedEOF
		if n == 0 {
			break
		}
		ref, err := post(ctx, s, buf[:n])
		if err != nil {
			return nil, err
		}
		indexes, counts, err = addToIndexes(ctx, s, indexes, counts, *ref)
		if err != nil {
			return nil, err
		}
	}
	root, err := finishIndexes(ctx, s, indexes, counts)
	if err != nil {
		return nil, err
	}
	return &File{
		Size: uint64(size),
		Root: *root,
	}, nil
}

func addToIndexes(ctx context.Context, s blobs.Poster, indexes []Index, counts []int, ref Ref) ([]Index, []int, error) {
	for i := 0; true; i++ {
		if len(indexes) <= i {
			indexes = append(indexes, newIndex())
			counts = append(counts, 0)
		}
		indexes[i].Set(counts[i], ref)
		counts[i]++
		if counts[i] < BranchingFactor {
			break
		} else {
			ref2, err := post(ctx, s, indexes[i].x)
			if err != nil {
				return nil, nil, err
			}
			counts[i] = 0
			ref = *ref2
		}
	}
	return indexes, counts, nil
}

func finishIndexes(ctx context.Context, s blobs.Poster, indexes []Index, counts []int) (*Ref, error) {
	for i := 0; i < len(indexes); i++ {
		if len(indexes)-1 == i && counts[i] == 1 {
			ref := indexes[i].Get(0)
			return &ref, nil
		}
		ref, err := post(ctx, s, indexes[i].x)
		if err != nil {
			return nil, err
		}
		if len(indexes) <= i+1 {
			indexes = append(indexes, newIndex())
			counts = append(counts, 0)
		}
		indexes[i+1].Set(counts[i+1], *ref)
		counts[i+1]++
	}
	return post(ctx, s, nil)
}

// Integer power: compute a**b using binary powering algorithm
// See Donald Knuth, The Art of Computer Programming, Volume 2, Section 4.6.3
func pow(a, b int) int {
	p := 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

func exp(x int) int {
	return pow(BranchingFactor, x)
}

func log2(x uint64) int {
	return 64 - bits.LeadingZeros64(x)
}

func depth(size uint64) int {
	if size <= blobs.MaxSize {
		return 0
	}
	return log2(size) / log2(BranchingFactor)
}

const BranchingFactor = blobs.MaxSize / RefSize

type Index struct {
	x []byte
}

func newIndex() Index {
	return Index{x: make([]byte, blobs.MaxSize)}
}

func newIndexUsing(x []byte) (Index, error) {
	if len(x) != blobs.MaxSize {
		return Index{}, errors.Errorf("data is not correct size for index")
	}
	return Index{x: x}, nil
}

func (idx Index) Get(i int) Ref {
	start := i * RefSize
	end := (i + 1) * RefSize
	ref, err := RefFromBytes(idx.x[start:end])
	if err != nil {
		panic(err)
	}
	return *ref
}

func (idx Index) Set(i int, ref Ref) {
	start := i * RefSize
	end := (i + 1) * RefSize
	buf := idx.x[start:end]
	copy(buf[:blobs.IDSize], ref.ID[:])
	copy(buf[blobs.IDSize:], ref.DEK[:])
}
