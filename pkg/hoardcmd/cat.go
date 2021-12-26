package hoardcmd

import (
	"encoding/hex"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:   "cat",
	Short: "write out the contents of an object to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("must provide fingerprint")
		}
		w := cmd.OutOrStdout()
		prefixHex := args[0]
		if len(prefixHex)%2 != 0 {
			prefixHex = prefixHex[:len(prefixHex)-1]
		}
		prefix, err := hex.DecodeString(prefixHex)
		if err != nil {
			return err
		}
		fps, err := h.ListByPrefix(ctx, prefix, 2)
		if err != nil {
			return err
		}
		if len(fps) == 0 {
			return errors.Errorf("not found")
		}
		if len(fps) > 1 {
			return errors.Errorf("prefix is non-specific. try a longer one.")
		}
		r, err := h.Get(ctx, fps[0])
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	},
}
