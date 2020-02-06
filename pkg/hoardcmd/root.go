package hoardcmd

import (
	"context"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/spf13/cobra"
)

var (
	h          *hoard.Node
	ctx        = context.Background()
	dataDir    string
	contentDir string
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&dataDir, "data-dir", "./", "--data-dir=/path/to/data")
	rootCmd.Flags().StringVar(&contentDir, "content-dir", "", "--content-dir=/path/to/content")
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
}

func setup() (err error) {
	if dataDir == "" {
		dataDir = "./"
	}
	sourcePaths := []string{}
	if contentDir != "" {
		sourcePaths = append(sourcePaths, contentDir)
	}
	params, err := hoard.DefaultParams(dataDir, sourcePaths)
	if err != nil {
		return err
	}
	h, err = hoard.New(params)
	return err
}
