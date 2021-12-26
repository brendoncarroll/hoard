package filecell

import (
	"testing"

	"github.com/brendoncarroll/go-state/cells"
	"github.com/brendoncarroll/go-state/cells/celltest"
	"github.com/brendoncarroll/go-state/posixfs"
)

func TestFileCell(t *testing.T) {
	celltest.CellTestSuite(t, func(t testing.TB) cells.Cell {
		fs := posixfs.NewDirFS(t.TempDir())
		return New(fs, "CELL")
	})
}
