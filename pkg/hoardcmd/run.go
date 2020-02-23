package hoardcmd

import (
	"net/http"

	"github.com/brendoncarroll/hoard/pkg/hoardhttp"
	"github.com/spf13/cobra"
)

var (
	uiAddr string
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&uiAddr, "ui-addr", "127.0.0.1:6026", "--ui-addr=192.168.1.100:8080")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the hoard server (includes web UI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		s := hoardhttp.New(h, uiDir)
		return http.ListenAndServe(uiAddr, s)
	},
}
