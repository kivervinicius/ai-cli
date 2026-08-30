package store

import (
	"errors"
	"testing"
)

func TestValidateExpectedPlanRevisionRejectsStaleRevision(t *testing.T) {
	err := validateExpectedPlanRevision(2, 3)
	if !errors.Is(err, ErrPlanRevisionConflict) {
		t.Fatalf("expected ErrPlanRevisionConflict, got %v", err)
	}
}

func TestValidateExpectedPlanRevisionAcceptsCurrentRevision(t *testing.T) {
	if err := validateExpectedPlanRevision(3, 3); err != nil {
		t.Fatalf("expected matching revision to pass, got %v", err)
	}
}
