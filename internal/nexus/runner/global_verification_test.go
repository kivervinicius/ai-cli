package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type globalRepairExecutor struct {
	fakeExecutor
	executions int
}

func (e *globalRepairExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	e.executions++
	if e.executions > 1 {
		if err := os.WriteFile(filepath.Join(run.Workspace, "global-ready"), []byte("verified\n"), 0o600); err != nil {
			return ExecutionResult{}, err
		}
	}
	return e.fakeExecutor.Execute(ctx, run, pkg, prompt)
}

func TestMissionRunnerGlobalDefinitionOfDoneReopensAfterFailure(t *testing.T) {
	workspace := t.TempDir()
	exec := &globalRepairExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"test -f global-ready"}
	contract.MaxNoProgress = 2
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "global", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}}}, workspace, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	var globalFailure bool
	for i := 0; i < 30 && run.State != StateCompletedVerified; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if stepErr != nil && len(run.GlobalVerifications) > 0 {
			globalFailure = true
		}
	}
	if !globalFailure {
		t.Fatal("expected global verification failure evidence")
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected convergence after global repair, got %s", run.State)
	}
	if len(run.GlobalVerifications) < 2 {
		t.Fatalf("expected failed and passing global verification evidence, got %+v", run.GlobalVerifications)
	}
	if run.GlobalVerifications[0].Passed || !run.GlobalVerifications[len(run.GlobalVerifications)-1].Passed {
		t.Fatalf("expected global verification to fail then pass: %+v", run.GlobalVerifications)
	}
}
