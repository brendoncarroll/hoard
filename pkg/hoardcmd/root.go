package hoardcmd

import (
	"context"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/spf13/cobra"
)

var (
	h   *hoard.Node
	ctx = context.Background()
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
}

var rootCmd = &cobra.Command{
	Use:   "hoard",
	Short: "Hoard is a CMS for data hoarders",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return setup()
	},
}

func setup() (err error) {
	params, err := hoard.DefaultParams("./")
	if err != nil {
		return err
	}
	h, err = hoard.New(params)
	return err
}
