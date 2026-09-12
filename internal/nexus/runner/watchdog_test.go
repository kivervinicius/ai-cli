package runner

import (
	"testing"
	"time"
)

func TestProgressWatchdogIgnoresFreshExecutingWork(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_fresh",
		State:          StateExecuting,
		StartedAt:      now,
		LastProgressAt: now,
		Contract:       AutonomyContract{StallTimeoutSeconds: 15},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting}},
	}
	if ApplyProgressWatchdog(run, now) {
		t.Fatal("fresh executing work must not be treated as stalled")
	}
	if run.State != StateExecuting {
		t.Fatalf("state mutated: %s", run.State)
	}
}

func TestProgressWatchdogFailsClosedOnStallWithoutNeedsUser(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_stall",
		State:          StateExecuting,
		StartedAt:      now.Add(-time.Hour),
		LastProgressAt: now.Add(-20 * time.Minute),
		UpdatedAt:      now, // lease/heartbeat activity is not progress
		Contract:       AutonomyContract{StallTimeoutSeconds: 900, EscalateOnFailure: true},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting}},
	}
	if !ApplyProgressWatchdog(run, now) {
		t.Fatal("stalled EXECUTING mission must fail closed")
	}
	if run.State != StateFailedNoProgress {
		t.Fatalf("expected FAILED_NO_PROGRESS, got %s", run.State)
	}
	if run.NeedsHuman != nil {
		t.Fatalf("stall must not become NEEDS_YOU: %+v", run.NeedsHuman)
	}
	if ClassifyAttention(run) == AttentionRequireUser {
		t.Fatal("FAILED_NO_PROGRESS must not classify as REQUIRE_USER")
	}
}

func TestProgressWatchdogDoesNotTouchTerminalStates(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_done",
		State:          StateCompletedVerified,
		LastProgressAt: now.Add(-time.Hour),
		Contract:       AutonomyContract{StallTimeoutSeconds: 1},
	}
	if ApplyProgressWatchdog(run, now) {
		t.Fatal("terminal runs must not be stalled")
	}
}
