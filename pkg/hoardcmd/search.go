package hoardcmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/brendoncarroll/hoard/pkg/labels"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "search for content by tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		pred, err := parsePredicate(args)
		if err != nil {
			return err
		}
		q := labels.Query{
			Where: *pred,
			Limit: 100,
		}
		logrus.Infof("searching for query %v\n", q)
		res, err := h.Search(ctx, q)
		if err != nil {
			return err
		}
		w := bufio.NewWriter(cmd.OutOrStdout())
		for _, id := range res {
			tags, err := h.GetLabels(ctx, id, "")
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "%v\t%v\n", id.String()[:8], tags); err != nil {
				return err
			}
		}
		return w.Flush()
	},
}

func parsePredicate(args []string) (*labels.Predicate, error) {
	var subQueries []labels.Query
	for _, arg := range args {
		switch {
		case strings.Contains(arg, "="):
			parts := strings.SplitN(arg, "=", 2)
			q := labels.Query{
				Where: labels.Predicate{
					Op:    labels.OpEq,
					Key:   parts[0],
					Value: parts[1],
				},
				Limit: 100,
			}
			subQueries = append(subQueries, q)
		default:
			return nil, errors.Errorf("could not parse into predicate %q", arg)
		}
	}
	return &labels.Predicate{
		Op:         labels.OpOR,
		SubQueries: subQueries,
	}, nil
}
