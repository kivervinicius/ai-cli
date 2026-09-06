package store

import "errors"

var ErrPlanRevisionConflict = errors.New("plan revision conflict")

func validateExpectedPlanRevision(expected, current int) error {
	if expected != current {
		return ErrPlanRevisionConflict
	}
	return nil
}
