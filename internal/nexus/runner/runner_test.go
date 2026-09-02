package runner

import (
	"context"
	"reflect"
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

func TestMissionRunnerPreservesFlowStepExecutionContracts(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	spec := PackageSpec{
		ID: "step", Title: "Step", Goal: "Goal", Role: "tester",
		AssignmentStrategy: "AUTO", ResourcePolicy: "PRESERVE_QUOTA", Provider: "codex", Profile: "fast",
		MaestroSkills: []string{"verification"}, RelevantPaths: []string{"internal/nexus"},
		VerificationRequirements: []string{"echo step-specific"}, AcceptanceCriteria: []string{"verified"},
	}
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "flow", ProjectID: "p", Revision: 2, Packages: []PackageSpec{spec}}, t.TempDir(), contract, "legacy-default")
	if err != nil {
		t.Fatal(err)
	}
	pkg := run.PackageRuns[0]
	if pkg.AssignedAgent != "" {
		t.Fatalf("AUTO must not inherit legacy default Agent: %q", pkg.AssignedAgent)
	}
	if pkg.AssignmentStrategy != "AUTO" || pkg.ResourcePolicy != "PRESERVE_QUOTA" || pkg.Provider != "codex" || pkg.Profile != "fast" {
		t.Fatalf("flow execution contract lost: %+v", pkg)
	}
	if !reflect.DeepEqual(pkg.MaestroSkills, []string{"verification"}) || !reflect.DeepEqual(pkg.RelevantPaths, []string{"internal/nexus"}) || !reflect.DeepEqual(pkg.VerificationRequirements, []string{"echo step-specific"}) {
		t.Fatalf("bounded step context lost: %+v", pkg)
	}
}

func TestMissionRunnerUsesStepVerificationRequirementsBeforeGlobalCommands(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"false"}
	plan := PlanSpec{ID: "flow", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal", VerificationRequirements: []string{"echo step-ok"}}}}
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
		t.Fatalf("step-specific verification should allow completion, got %s", run.State)
	}
	if got := run.PackageRuns[0].Verifications; len(got) == 0 || got[0].Command != "echo step-ok" {
		t.Fatalf("wrong verification evidence: %+v", got)
	}
}
