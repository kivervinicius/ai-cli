package nexus

import (
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/contextsnapshot"
)

func TestDecideIntentRoutesAtomicFixDirectly(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	if decision.Strategy != IntentDirect || decision.RecommendedAction != "execute_single_work_unit" {
		t.Fatalf("unexpected direct decision: %+v", decision)
	}
}

func TestDecideIntentUsesProjectFactsBeforeClarifying(t *testing.T) {
	snapshot := &contextsnapshot.ProjectContextSnapshot{
		Completeness: contextsnapshot.CompletenessComplete,
		Identity:     contextsnapshot.CodeIdentity{IdentityDigest: "sha"},
		Facts: []contextsnapshot.ProjectFact{{
			Key: "tests.present", Value: true,
		}},
	}
	decision := DecideIntent("quero melhorar os testes automatizados deste projeto", snapshot)
	if decision.Strategy == IntentClarify {
		t.Fatalf("known repository facts should prevent ritual clarification: %+v", decision)
	}
	if len(decision.KnownFacts) != 1 || decision.SnapshotIdentity != "sha" {
		t.Fatalf("expected bounded fact grounding: %+v", decision)
	}
}

func TestDecideIntentRoutesCompleteMultiStepGoalToPlan(t *testing.T) {
	decision := DecideIntent("implemente a próxima etapa com testes, revisão e critérios de aceitação", nil)
	if decision.Strategy != IntentPlan || decision.RecommendedAction != "create_work_plan" {
		t.Fatalf("unexpected plan decision: %+v", decision)
	}
}

func TestDecideIntentOnlyClarifiesMaterialMissingObjective(t *testing.T) {
	decision := DecideIntent("construa", nil)
	if decision.Strategy != IntentClarify || len(decision.BlockingQuestions) != 1 {
		t.Fatalf("expected one material clarification: %+v", decision)
	}
}

func TestPersistIntentDecisionFactsUsesExistingWorkPlanMetadata(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	facts, err := PersistIntentDecisionFacts(map[string]string{"framework": "go"}, decision)
	if err != nil {
		t.Fatal(err)
	}
	if facts[IntentDecisionFactKey] == "" || facts[IntentStrategyFactKey] != string(IntentDirect) || facts[IntentConfidenceFactKey] != "HIGH" {
		t.Fatalf("intent decision metadata was not persisted: %#v", facts)
	}
	if facts["framework"] != "go" {
		t.Fatalf("existing structured facts were lost: %#v", facts)
	}
}
