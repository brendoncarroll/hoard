package hoardcmd

import (
	"bufio"
	"fmt"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/spf13/cobra"
)

var lsIDCmd = &cobra.Command{
	Use:   "ls-id",
	Short: "lists expressions by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEachExpr(ctx, hoard.IDSpan{}, func(id hoard.ID, _ hoard.Expr) error {
			fmt.Fprintf(w, "%v\n", id)
			return nil
		}); err != nil {
			return nil
		}
		return w.Flush()
	},
}

var lsExprCmd = &cobra.Command{
	Use:   "ls",
	Short: "lists expressions and their labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := bufio.NewWriter(cmd.OutOrStdout())
		if err := h.ForEachExpr(ctx, hoard.IDSpan{}, func(id hoard.ID, e hoard.Expr) error {
			ls, err := h.GetLabels(ctx, id, "")
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%v\n", id)
			for _, pair := range ls {
				fmt.Fprintf(w, "\t%v\n", pair)
			}
			return nil
		}); err != nil {
			return nil
		}
		return w.Flush()
	},
}
