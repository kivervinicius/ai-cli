package cooldown

import (
	"testing"
	"time"
)

func TestCooldownTracker(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	tracker := NewTracker()

	// Initially not rate limited
	isLimited, _ := tracker.IsRateLimited("codex", "work")
	if isLimited {
		t.Fatalf("expected not rate limited initially")
	}

	// Record rate limit of 1 hour
	tracker.RecordRateLimit("codex", "work", time.Hour, nil, "HTTP 429 quota exceeded")
	isLimited, rec := tracker.IsRateLimited("codex", "work")
	if !isLimited || rec == nil {
		t.Fatalf("expected profile to be rate limited")
	}
	if rec.Reason != "HTTP 429 quota exceeded" {
		t.Fatalf("unexpected reason: %s", rec.Reason)
	}

	// Record expired rate limit
	past := time.Now().Add(-10 * time.Minute)
	tracker.RecordRateLimit("codex", "personal", time.Minute, &past, "Old limit")
	isLimited2, _ := tracker.IsRateLimited("codex", "personal")
	if isLimited2 {
		t.Fatalf("expected expired limit to not be active")
	}

	// Test Clear
	tracker.Clear("codex", "work")
	isLimited3, _ := tracker.IsRateLimited("codex", "work")
	if isLimited3 {
		t.Fatalf("expected cleared profile to not be rate limited")
	}
}
