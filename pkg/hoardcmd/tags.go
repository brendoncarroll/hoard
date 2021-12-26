package hoardcmd

import (
	"bufio"
	"fmt"

	"github.com/spf13/cobra"
)

var lsTagsCmd = &cobra.Command{
	Use:   "ls-tags",
	Short: "list tags to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEachTagKey(ctx, func(k string) error {
			_, err := fmt.Fprintf(w, "%s\n", k)
			return err
		}); err != nil {
			return err
		}
		return w.Flush()
	},
}

var lsTagValuesCmd = &cobra.Command{
	Use:   "ls-values",
	Short: "list tag values to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tagKey := args[0]
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEachTagValue(ctx, tagKey, func(v []byte) error {
			_, err := fmt.Fprintf(w, "%q\n", v)
			return err
		}); err != nil {
			return err
		}
		return w.Flush()
	},
}
