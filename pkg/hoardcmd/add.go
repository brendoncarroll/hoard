package hoardcmd

import (
	"fmt"

	"github.com/brendoncarroll/go-state/posixfs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds a file or every file in a directory individually",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		fs := posixfs.NewOSFS()
		w := cmd.OutOrStdout()
		logrus.Infof("importing %s ...\n", target)
		err := posixfs.WalkLeaves(ctx, fs, target, func(p string, de posixfs.DirEnt) error {
			f, err := fs.OpenFile(p, posixfs.O_RDONLY, 0)
			if err != nil {
				return err
			}
			defer f.Close()
			fp, err := h.Add(ctx, f)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%v %s\n", fp, p)
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	},
}
