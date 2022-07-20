package labels

import "github.com/pkg/errors"

func errInvalidOp(op PredicateOp) error {
	return errors.Errorf("invalid predicate op %v", op)
}
