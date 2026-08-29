package runner

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestMissionRunner_Lifecycle(t *testing.T) {
	runner := NewMissionRunner()
	tmpDir := t.TempDir()

	plan := store.WorkPlan{
		ID:        "plan-1",
		ProjectID: "proj-1",
		Title:     "Auth Implementation",
		Phases: []store.PlanPhase{
			{
				ID:    "phase-1",
				Title: "Phase 1",
				Packages: []store.WorkPackage{
					{
						ID:       "pkg-1",
						Title:    "Unit Tests Setup",
						Goal:     "Write initial unit tests",
						Priority: "CRITICAL",
					},
				},
			},
		},
	}

	contract := DefaultAutonomyContract()
	// Use echo command for deterministic pass in test environment
	contract.VerificationCommands = []string{"echo OK"}

	run, err := runner.StartMissionRun(context.Background(), plan, tmpDir, contract, "agt-1")
	if err != nil {
		t.Fatalf("StartMissionRun failed: %v", err)
	}

	if run.State != StateExecuting {
		t.Errorf("expected state EXECUTING, got %s", run.State)
	}

	// Step through the state machine until completion
	done := false
	for i := 0; i < 10; i++ {
		completed, err := runner.ExecuteNextStep(context.Background(), run, tmpDir)
		if err != nil {
			t.Fatalf("step %d failed: %v", i, err)
		}
		if completed {
			done = true
			break
		}
	}

	if !done {
		t.Errorf("expected mission run to complete, final state: %s", run.State)
	}
}

func TestMissionRunner_BoundedRetries(t *testing.T) {
	runner := NewMissionRunner()
	tmpDir := t.TempDir()

	plan := store.WorkPlan{
		ID:        "plan-fail",
		ProjectID: "proj-1",
		Title:     "Failing Job",
		Phases: []store.PlanPhase{
			{
				ID: "phase-1",
				Packages: []store.WorkPackage{
					{
						ID:       "pkg-fail",
						Title:    "Always Fails",
						Goal:     "Trigger retry exhaustion",
						Priority: "HIGH",
					},
				},
			},
		},
	}

	contract := DefaultAutonomyContract()
	contract.MaxRetries = 2
	contract.VerificationCommands = []string{"false"} // Always fails

	run, err := runner.StartMissionRun(context.Background(), plan, tmpDir, contract, "agt-1")
	if err != nil {
		t.Fatalf("StartMissionRun failed: %v", err)
	}

	var finalErr error
	for i := 0; i < 15; i++ {
		completed, err := runner.ExecuteNextStep(context.Background(), run, tmpDir)
		if err != nil {
			finalErr = err
			break
		}
		if completed {
			break
		}
	}

	if finalErr == nil {
		t.Errorf("expected run to fail after exceeding max retries")
	}

	if run.State != StateEscalated && run.State != StateFailed {
		t.Errorf("expected run state ESCALATED or FAILED, got %s", run.State)
	}
}
