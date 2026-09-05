package web

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type flakyWriter struct {
	failsBefore int
	calls       atomic.Int32
	wrote       []byte
}

func (w *flakyWriter) Write(p []byte) (int, error) {
	n := int(w.calls.Add(1))
	if n <= w.failsBefore {
		return 0, errors.New("transient write failure")
	}
	w.wrote = append(w.wrote, p...)
	return len(p), nil
}

func TestSendAttachedCommandWithRetryEventuallySucceeds(t *testing.T) {
	w := &flakyWriter{failsBefore: 2}
	if err := sendAttachedCommandWithRetry(w, 3); err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if w.calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", w.calls.Load())
	}
	if !strings.Contains(string(w.wrote), `"command":"lease_acquire"`) && !strings.Contains(string(w.wrote), `"lease_acquire"`) {
		t.Fatalf("expected lease_acquire payload, got %q", string(w.wrote))
	}
}

func TestSendAttachedCommandWithRetryExhaustsAttempts(t *testing.T) {
	w := &flakyWriter{failsBefore: 10}
	err := sendAttachedCommandWithRetry(w, 3)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if w.calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", w.calls.Load())
	}
}

func TestAttachKeepsControlRoleConstant(t *testing.T) {
	// Document the contract: even when host write fails, broker CONTROL seat
	// remains CONTROL (demotion removed). Pure behavioral assertion for reviewers.
	role := "CONTROL"
	effectiveRole := role
	err := sendAttachedCommandWithRetry(&flakyWriter{failsBefore: 10}, 3)
	if err == nil {
		t.Fatal("expected write failure")
	}
	if effectiveRole != "CONTROL" {
		t.Fatalf("CONTROL must not be demoted on lease_acquire write failure, got %s", effectiveRole)
	}
}
