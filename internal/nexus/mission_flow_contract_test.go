package nexus

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestPlanToRunnerSpecCarriesFlowStepContracts(t *testing.T) {
	plan := &store.WorkPlan{ID: "plan", ProjectID: "project", CurrentRevision: 3, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{
		ID: "step", Title: "Step", Goal: "Goal", Priority: "HIGH", Role: "tester", AssignmentStrategy: "AUTO", ResourcePolicy: "PRESERVE_QUOTA", Provider: "codex", Profile: "fast",
		MaestroSkills: []string{"verification"}, RelevantPaths: []string{"internal"}, AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"go test ./..."},
	}}}}}
	spec, err := planToRunnerSpec(plan, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	got := spec.Packages[0]
	if got.AssignmentStrategy != "AUTO" || got.ResourcePolicy != "PRESERVE_QUOTA" || got.Provider != "codex" || got.Profile != "fast" {
		t.Fatalf("assignment/resource contract lost: %+v", got)
	}
	if !reflect.DeepEqual(got.MaestroSkills, []string{"verification"}) || !reflect.DeepEqual(got.RelevantPaths, []string{"internal"}) || !reflect.DeepEqual(got.VerificationRequirements, []string{"go test ./..."}) {
		t.Fatalf("context/verification contract lost: %+v", got)
	}
}

func TestValidateFlowExecutionContractRejectsContradictoryAssignment(t *testing.T) {
	plan := store.WorkPlan{ID: "plan", ProjectID: "project", Phases: []store.PlanPhase{{ID: "p", Packages: []store.WorkPackage{{ID: "A", AssignmentStrategy: "CREATE", AgentAllocation: "agent-1"}}}}}
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("CREATE + existing Agent must be rejected")
	}
	plan.Phases[0].Packages[0].AssignmentStrategy = "EXISTING"
	plan.Phases[0].Packages[0].AgentAllocation = ""
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("EXISTING without Agent must be rejected")
	}
	plan.Phases[0].Packages[0].AssignmentStrategy = "AUTO"
	plan.Phases[0].Packages[0].ResourcePolicy = "MANUAL"
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("MANUAL resource policy without provider/profile must be rejected")
	}
}

func TestStartMissionRunAutonomousRejectsMissingWorktreeAdmission(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Strict admission", CanonicalPath: t.TempDir(), DefaultIsolation: "project"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase", Packages: []store.WorkPackage{{Title: "Required worktree", Goal: "Run safely", Status: "READY", Role: "implementer", AcceptanceCriteria: []string{"done"}}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = n.StartMissionRun(context.Background(), plan.ID, "", runner.DefaultAutonomyContract(), true)
	if err == nil {
		t.Fatal("expected autonomous mission admission to reject project checkout")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "worktree") {
		t.Fatalf("expected actionable worktree admission error, got %v", err)
	}
}

func TestNorthStar_PromptToFlowToPreflightCycle(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "NorthStarE2E", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Decompose prompt into flow proposal
	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID:    project.ID,
		Goal:         "Implement responsive layout and run automated regression tests",
		SourcePrompt: "Ensure container queries adapt smoothly and unit tests pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Flow.Steps) < 2 {
		t.Fatalf("expected structured multi-step flow, got %d steps", len(proposal.Flow.Steps))
	}

	// 2. Materialize Flow into durable WorkPlan
	plan := WorkPlanFromFlow(proposal.Flow)
	createdPlan, err := st.CreateWorkPlan(plan)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Run Preflight check before execution
	report, err := n.PreflightFlow(context.Background(), createdPlan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatalf("expected preflight to pass for valid generated flow, got %+v", report)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected preflight checks in report")
	}
}
