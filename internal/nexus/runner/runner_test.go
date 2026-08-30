package runner

import (
	"context"
	"testing"
)

func TestMissionRunnerLifecycleCompletesOnlyAfterVerificationAndReview(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"echo OK"}
	plan := PlanSpec{ID: "plan-1", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{{ID: "pkg-1", Title: "Unit Tests", Goal: "Write tests"}}}

	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected completed verified, got %s", run.State)
	}
	if len(run.PackageRuns[0].Verifications) == 0 {
		t.Fatal("verification evidence missing")
	}
	if len(run.PackageRuns[0].Verdicts) == 0 {
		t.Fatal("review evidence missing")
	}
}

func TestMissionRunnerBoundedRetries(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 2
	contract.VerificationCommands = []string{"false"}
	plan := PlanSpec{ID: "plan-fail", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{{ID: "pkg-fail", Title: "Always fails", Goal: "Fail"}}}

	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	var finalErr error
	for i := 0; i < 20; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if stepErr != nil && run.PackageRuns[0].State == StateFailed {
			finalErr = stepErr
			break
		}
	}
	if finalErr == nil {
		t.Fatal("expected bounded retry failure")
	}
	if run.State != StateBlockedNeedsUser && run.State != StateFailedNoProgress {
		t.Fatalf("unexpected run state %s", run.State)
	}
}

func TestMissionRunnerRunsFinalVerificationAfterIndependentReview(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxTotalIterations = 30
	contract.VerificationCommands = []string{"echo verified"}
	plan := PlanSpec{ID: "plan-final-verify", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}

	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	seenVerifying := false
	for i := 0; i < 20 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if run.PackageRuns[0].State == StateVerifying {
			seenVerifying = true
		}
	}
	if !seenVerifying {
		t.Fatal("review approval must transition through VERIFYING before VERIFIED")
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("state=%s", run.State)
	}
	if got := len(run.PackageRuns[0].Verifications); got < 2 {
		t.Fatalf("expected test verification plus final verification evidence, got %d", got)
	}
}
