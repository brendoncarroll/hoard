package hoardcmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(runCmd)

}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the hoard server (includes web UI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			laddr string
		)
		cmd.Flags().StringVar(&laddr, "ui-addr", "127.0.0.1:6026", "--ui-addr=192.168.1.100:8080")
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return h.Serve(ctx, laddr)
	},
}
