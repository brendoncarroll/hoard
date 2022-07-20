package hcorpus

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	ctx := context.Background()
	op, s := setup(t)
	root, err := op.NewEmpty(ctx, s)
	require.NoError(t, err)
	expectedData := "my test string"
	const N = 5
	for i := 0; i < N; i++ {
		fp, root, err := op.Post(ctx, s, *root, []byte(expectedData))
		require.NoError(t, err)
		t.Log(fp)
		data, err := op.Get(ctx, s, *root, fp)
		require.NoError(t, err)
		require.Equal(t, expectedData, string(data))
	}
}

func setup(t testing.TB) (*Operator, cadata.Store) {
	op := New()
	s := cadata.NewMem(cadata.DefaultHash, gotfs.DefaultMaxBlobSize)
	return op, s
}
