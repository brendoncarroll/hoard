package hoardcmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(addFileCmd)
}

var addFileCmd = &cobra.Command{
	Use:   "add-file",
	Short: "Adds a file or every file in a directory individually",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := args[0]
		err := h.AddAllFiles(ctx, p)
		if err != nil {
			return err
		}
		return nil
	},
}
