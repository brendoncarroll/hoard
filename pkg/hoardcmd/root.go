package hoardcmd

import (
	"context"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/go-state/cells"
	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/spf13/cobra"
)

var (
	ctx = context.Background()
	h   *hoard.Hoard
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(catCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsTagsCmd)
}

var rootCmd = &cobra.Command{
	Use:   "hoard",
	Short: "Hoard is a CMS for data hoarders",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		return setup()
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return teardown()
	},
}

func setup() (err error) {
	newCell := func() cells.Cell {
		return cells.NewMem(1 << 16)
	}
	newStore := func() cadata.Store {
		return cadata.NewMem(cadata.DefaultHash, gotfs.DefaultMaxBlobSize)
	}
	h = hoard.New(hoard.Params{
		Corpus: hoard.Volume{
			Cell:  newCell(),
			Store: newStore(),
		},
		Index: hoard.Volume{
			Cell:  newCell(),
			Store: newStore(),
		},
	})
	return nil
}

func teardown() error {
	return nil
}
