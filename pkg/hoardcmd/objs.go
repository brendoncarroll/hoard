package hoardcmd

import (
	"bufio"
	"fmt"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/brendoncarroll/hoard/pkg/tagging"
	"github.com/spf13/cobra"
)

var lsFPCmd = &cobra.Command{
	Use:   "ls-fp",
	Short: "lists objects by their fingerprint",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEach(ctx, func(id hoard.OID, _ []tagging.Tag) error {
			fmt.Fprintf(w, "%v\n", id)
			return nil
		}); err != nil {
			return nil
		}
		return w.Flush()
	},
}

var lsObjsCmd = &cobra.Command{
	Use:   "ls",
	Short: "lists objects and their tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEach(ctx, func(id hoard.OID, tags []tagging.Tag) error {
			fmt.Fprintf(w, "%v\n", id)
			for _, tag := range tags {
				fmt.Fprintf(w, "\t%v\n", tag)
			}
			return nil
		}); err != nil {
			return nil
		}
		return w.Flush()
	},
}
