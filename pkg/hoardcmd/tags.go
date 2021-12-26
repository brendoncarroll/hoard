package hoardcmd

import (
	"bufio"
	"fmt"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/brendoncarroll/hoard/pkg/tagging"
	"github.com/spf13/cobra"
)

var lsTagsCmd = &cobra.Command{
	Use:   "ls-tags",
	Short: "list tags to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		fmtStr := "%-64v\t%-20s\t%-30s\n"
		if _, err := fmt.Fprintf(w, fmtStr, "FINGERPRINT", "KEY", "VALUE"); err != nil {
			return err
		}
		if err := h.ListTags(ctx, func(id hoard.OID, tag tagging.Tag) error {
			_, err := fmt.Fprintf(w, fmtStr, id, tag.Key, tag.Value)
			return err
		}); err != nil {
			return err
		}
		return w.Flush()
	},
}
