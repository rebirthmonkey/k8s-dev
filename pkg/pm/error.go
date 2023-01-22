package pm

import (
	"gopkg.in/multierror.v1"
)

func multiErrors(e1 error, e2 error) error {
	switch {
	case e1 == nil && e2 == nil:
		return nil
	case e1 == nil:
		return e2
	case e2 == nil:
		return e1
	default:
		merrs1, ok := e1.(multierror.MultipleErrors)
		if !ok {
			merrs1 = []error{e1}
		}
		merrs2, ok := e2.(multierror.MultipleErrors)
		if !ok {
			merrs2 = []error{e2}
		}
		var merrs []error
		merrs = append(merrs, merrs1)
		merrs = append(merrs, merrs2)
		return multierror.MultipleErrors(merrs)
	}
}

type nonFatalError struct {
	s string
}

func (n *nonFatalError) Error() string {
	return n.s
}
