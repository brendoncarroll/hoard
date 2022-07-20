package hoardcmd

import (
	"context"
	"path/filepath"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/go-state/cadata/fsstore"
	"github.com/brendoncarroll/go-state/posixfs"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/spf13/cobra"

	"github.com/brendoncarroll/hoard/pkg/filecell"
	"github.com/brendoncarroll/hoard/pkg/hoard"
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
	rootCmd.AddCommand(lsKeysCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(lsIDCmd)
	rootCmd.AddCommand(lsExprCmd)
	rootCmd.AddCommand(lsTagValuesCmd)
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
	dir, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	workingDir := posixfs.NewDirFS(dir)
	if err := posixfs.MkdirAll(workingDir, "hoard_data/blobs", 0o755); err != nil {
		return err
	}
	cell := filecell.New(workingDir, "hoard_data/CELL")
	storeFS := posixfs.NewPrefixed(workingDir, "hoard_data/blobs")
	store := fsstore.New(storeFS, cadata.DefaultHash, gotfs.DefaultMaxBlobSize)
	h = hoard.New(hoard.Params{
		Volume: hoard.Volume{
			Cell:   cell,
			Corpus: store,
			Index:  store,
			GLFS:   store,
		},
	})
	return nil
}

func teardown() error {
	return nil
}

type Config struct {
	Corpus hoard.VolumeSpec `json:"corpus"`
	Index  hoard.VolumeSpec `json:"index"`

	Indexes hoard.VolumeSpec `json:"indexes"`
}
