package nexus

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func testAccounts() []ProviderAccount {
	return []ProviderAccount{
		{Provider: "agy", Profile: "frontend", Authenticated: true, Available: true, QuotaRemaining: 0.8},
		{Provider: "opencode", Profile: "qa", Authenticated: true, Available: true, QuotaRemaining: 0.7},
	}
}

func TestRuntimeRoutingDecisionPersistsExplainableSelectedFields(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["selected_provider"] != "agy" || payload["selected_profile"] != "frontend" || payload["selected_model"] != "cheap" {
		t.Fatalf("decision omitted selected resource projection: %s", raw)
	}
	if _, ok := payload["affinity_policy"]; !ok {
		t.Fatalf("decision omitted affinity policy: %s", raw)
	}
}

func TestMissionRoutingProjectionPersistsActualSelectionAndAffinity(t *testing.T) {
	decision := buildMissionRoutingDecision(
		&runner.PackageRun{PackageID: "pkg-1"},
		store.Agent{ID: "agent-1"},
		TaskRequirements{TaskKind: "coding"},
		ProviderAccount{Provider: "opencode", Profile: "qa", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "mimo"},
		"agy",
		"frontend",
		AffinityPrefer,
		true,
		"preferred resource unavailable; scheduler selected a healthy alternative",
	)
	if decision.AffinityPolicy.Provider.Value != "agy" || decision.AffinityPolicy.Profile.Value != "frontend" {
		t.Fatalf("affinity policy was not persisted: %+v", decision.AffinityPolicy)
	}
	if decision.SelectedProvider != "opencode" || decision.SelectedProfile != "qa" || decision.SelectedModel != "mimo" {
		t.Fatalf("actual selection was not projected: %+v", decision)
	}
	if !decision.Fallback || decision.Reason == "" {
		t.Fatalf("fallback explanation missing: %+v", decision)
	}
}

func testModels() []ModelCandidate {
	return []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "cheap", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "strong", CostRank: 3, ReasoningRank: 3, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "opencode", Profile: "qa", Model: "mimo", CostRank: 1, ReasoningRank: 2, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}
}

func TestRuntimeRoutingPreferFallsBackWithoutMutatingDesiredPolicy(t *testing.T) {
	policy := RuntimeAffinityPolicy{
		Provider: AffinityPreference{Mode: AffinityPrefer, Value: "missing"},
		Model:    AffinityPreference{Mode: AffinityPrefer, Value: "strong"},
	}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Fallback || decision.Actual.Provider != "agy" || decision.Actual.Model != "strong" {
		t.Fatalf("unexpected fallback decision: %+v", decision)
	}
	if decision.Desired.Provider.Value != "missing" || decision.Desired.Model.Value != "strong" {
		t.Fatalf("desired affinity was mutated: %+v", decision.Desired)
	}
}

func TestRuntimeRoutingPinFailsClosed(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPin, Value: "missing"}}
	_, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if !errors.Is(err, ErrBlockedResource) {
		t.Fatalf("expected blocked resource, got %v", err)
	}
}

func TestRuntimeRoutingUsesCheapCapableModelForSimpleTask(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Model != "cheap" {
		t.Fatalf("expected cheap capable model, got %+v", decision.Actual)
	}
}

func TestRuntimeRoutingRaisesReasoningFloorForArchitecture(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "architecture"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Model != "strong" {
		t.Fatalf("expected stronger model for architecture, got %+v", decision.Actual)
	}
}
