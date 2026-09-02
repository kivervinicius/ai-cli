package nexus

import (
	"reflect"
	"testing"

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
