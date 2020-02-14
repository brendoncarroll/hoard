package hoardcmd

import (
	"context"
	"path/filepath"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/spf13/cobra"
)

var (
	h          *hoard.Node
	ctx        = context.Background()
	dataDir    string
	contentDir string
	uiDir      string
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "./", "--data-dir=/path/to/data")
	rootCmd.PersistentFlags().StringVar(&contentDir, "content-dir", "", "--content-dir=/path/to/content")
	rootCmd.PersistentFlags().StringVar(&uiDir, "ui-dir", "", "--ui-dir=/path/to/ui")
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
	if uiDir != "" {
		uiDir, err = filepath.Abs(uiDir)
		if err != nil {
			return err
		}
	}

	params, err := hoard.DefaultParams(dataDir, sourcePaths, uiDir)
	if err != nil {
		return err
	}
	h, err = hoard.New(params)
	return err
}
