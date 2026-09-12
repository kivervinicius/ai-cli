package nexus

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/contextsnapshot"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	nexusskills "github.com/kivervinicius/ai-cli/internal/nexus/skills"
)

type localAutopilotExecutor struct{}

func (localAutopilotExecutor) Allocate(_ context.Context, run *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: "agent-local", Workspace: run.Workspace}, nil
}

func (localAutopilotExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "prompt-local", Content: "execute bounded local task"}, nil
}

func (localAutopilotExecutor) Execute(context.Context, *runner.MissionRun, *runner.PackageRun, string) (runner.ExecutionResult, error) {
	return runner.ExecutionResult{RuntimeID: "runtime-local"}, nil
}

func (localAutopilotExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-local", ReviewedAt: time.Now().UTC()}, nil
}

func TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := contextsnapshot.Discover(root, contextsnapshot.Metadata{ProjectID: "local-project"})
	if err != nil {
		t.Fatal(err)
	}
	decision := DecideIntent("corrija este teste quebrado", &snapshot)
	if decision.Strategy != IntentDirect {
		t.Fatalf("atomic local request must route directly: %+v", decision)
	}

	catalog, err := nexusskills.NewCatalog(nexusskills.NewBuiltinSource())
	if err != nil {
		t.Fatal(err)
	}
	if skill, ok := catalog.Resolve("testing"); !ok || skill.Source != nexusskills.SourceBuiltin {
		t.Fatalf("generic testing Skill was not resolved: %+v", skill)
	}

	account := ProviderAccount{Provider: "local", Profile: "test", Health: "healthy", Authenticated: true, Available: true}
	model, fallback, reason, err := ResolveTaskModel(TaskRequirements{TaskKind: "testing"}, AffinityPreference{Mode: AffinityAuto}, account, []ModelCandidate{{
		Provider: "local", Profile: "test", Model: "economy", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true,
	}})
	if err != nil || fallback || model.Model != "economy" || reason == "" {
		t.Fatalf("local task-aware model selection failed: model=%+v fallback=%t reason=%q err=%v", model, fallback, reason, err)
	}

	contract := runner.DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"true"}
	mission := runner.NewMissionRunner(runner.NewMemoryRunRepository(), localAutopilotExecutor{})
	run, err := mission.StartMissionRun(context.Background(), runner.PlanSpec{
		ID: "local-autopilot", ProjectID: "local-project", Revision: 1,
		Packages: []runner.PackageSpec{{
			ID: "test-fix", Title: "Fix test", Goal: "Fix the broken test", Role: "tester",
			SkillIDs: []string{"testing"}, AcceptanceCriteria: []string{"test passes"}, VerificationRequirements: []string{"true"},
		}},
	}, root, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := mission.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != runner.StateCompletedVerified || completed.PackageRuns[0].State != runner.StateVerified {
		t.Fatalf("local autopilot did not reach verified completion: state=%s package=%s", completed.State, completed.PackageRuns[0].State)
	}
}
