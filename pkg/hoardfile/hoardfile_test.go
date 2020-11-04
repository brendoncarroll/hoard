package hoardfile

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"testing"

	"github.com/blobcache/blobcache/pkg/blobs"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestCreateFile(t *testing.T) {
	ctx := context.Background()
	s := blobs.NewMem()

	const size = blobs.MaxSize * 3
	rng := rand.New(rand.NewSource(0))
	r := io.LimitReader(rng, size)
	f, err := Create(ctx, s, r)
	require.Nil(t, err)
	require.NotNil(t, f)
	require.Equal(t, uint64(size), f.Size)

	exists, err := s.Exists(ctx, f.Root.ID)
	require.Nil(t, err)
	require.True(t, exists)
	require.Equal(t, 4, s.Len())
}

func TestCreateRead(t *testing.T) {
	for _, size := range []int{
		0,
		1,
		100,
		blobs.MaxSize / 2,
		blobs.MaxSize,
		blobs.MaxSize * 2,
		blobs.MaxSize*2 - 1,
		blobs.MaxSize*2 + 1,
		blobs.MaxSize * BranchingFactor,
		blobs.MaxSize*BranchingFactor + 1,
		blobs.MaxSize*BranchingFactor - 1,
	} {
		t.Run(fmt.Sprintf("CreateRead-%d", size), func(t *testing.T) {
			testCreateRead(t, size)
		})
	}
}

func testCreateRead(t *testing.T, size int) {
	ctx := context.Background()
	s := blobs.NewMem()

	rng := rand.New(rand.NewSource(0))
	pr, pw := io.Pipe()
	h := fnv.New128()
	w := io.MultiWriter(h, pw)
	var file *File
	eg := errgroup.Group{}
	eg.Go(func() error {
		io.CopyN(w, rng, int64(size))
		pw.Close()
		return nil
	})
	eg.Go(func() error {
		f, err := Create(ctx, s, pr)
		file = f
		return err
	})
	require.Nil(t, eg.Wait())
	require.NotNil(t, file)

	r := NewReader(ctx, s, *file)
	h2 := fnv.New128()
	io.Copy(h2, r)

	require.Equal(t, h.Sum(nil), h2.Sum(nil))
}
