package maestrogates

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateStrictAllowsNoGatesWithoutMaestro(t *testing.T) {
	got, err := ValidateStrict(nil, false, nil, errors.New("unavailable"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no gates, got %v", got)
	}
}

func TestValidateStrictRejectsRequestedGatesWhenMaestroUnavailable(t *testing.T) {
	_, err := ValidateStrict([]string{"verification"}, false, nil, errors.New("binary missing"))
	if err == nil || !strings.Contains(err.Error(), "MAESTRO_DEGRADED") {
		t.Fatalf("expected MAESTRO_DEGRADED, got %v", err)
	}
}

func TestValidateStrictRejectsUnknownRequestedGate(t *testing.T) {
	_, err := ValidateStrict([]string{"security-audit"}, true, []string{"verification"}, nil)
	if err == nil || !strings.Contains(err.Error(), "security-audit") {
		t.Fatalf("expected unknown gate error, got %v", err)
	}
}

func TestValidateStrictPreservesRequestedGateOrder(t *testing.T) {
	got, err := ValidateStrict([]string{"verification", "refactoring"}, true, []string{"refactoring", "verification"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "verification" || got[1] != "refactoring" {
		t.Fatalf("unexpected gates %v", got)
	}
}
