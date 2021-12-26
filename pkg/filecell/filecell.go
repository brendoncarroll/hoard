package filecell

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"github.com/brendoncarroll/go-state/cells"
	"github.com/brendoncarroll/go-state/posixfs"
)

type Cell struct {
	fs posixfs.FS
	p  string
	mu sync.Mutex
}

func New(fs posixfs.FS, p string) *Cell {
	return &Cell{fs: fs, p: p}
}

func (c *Cell) CAS(ctx context.Context, actual, prev, next []byte) (bool, int, error) {
	if len(next) > c.MaxSize() {
		return false, 0, cells.ErrTooLarge{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := posixfs.ReadFile(ctx, c.fs, c.p)
	if err != nil && !os.IsNotExist(err) {
		return false, 0, err
	}
	var swapped bool
	if bytes.Equal(data, prev) {
		if err := posixfs.PutFile(ctx, c.fs, c.p, 0o644, bytes.NewReader(next)); err != nil {
			return false, 0, err
		}
		data = next
		swapped = true
	} else {
		swapped = false
	}
	if len(actual) < len(data) {
		return swapped, 0, io.ErrShortBuffer
	}
	return swapped, copy(actual, data), nil
}

func (c *Cell) Read(ctx context.Context, buf []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := posixfs.ReadFile(ctx, c.fs, c.p)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	return copy(buf, data), nil
}

func (c *Cell) MaxSize() int {
	return 1 << 16
}
