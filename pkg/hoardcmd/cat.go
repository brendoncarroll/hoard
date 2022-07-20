package hoardcmd

import (
	"encoding/hex"
	"io"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/gotvc/got/pkg/gotkv"
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
		span := hoard.IDSpan{}
		span = span.WithLowerIncl(cadata.IDFromBytes(prefix))
		span = span.WithUpperExcl(cadata.IDFromBytes(gotkv.PrefixEnd(prefix)))
		ids, err := h.ListIDs(ctx, span)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return errors.Errorf("not found")
		}
		if len(ids) > 1 {
			return errors.Errorf("prefix is non-specific. try a longer one.")
		}
		r, err := h.NewReader(ctx, ids[0])
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	},
}
