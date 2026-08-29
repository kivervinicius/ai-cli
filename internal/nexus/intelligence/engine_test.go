package intelligence

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	analysis *IntentAnalysis
	unknowns []AmbiguityItem
	outline  []WorkPackageOutline
	err      error
}

func (s *stubProvider) Name() string                   { return "stub" }
func (s *stubProvider) Available(context.Context) bool { return s.err == nil }
func (s *stubProvider) AnalyzeIntent(context.Context, string, map[string]any) (*IntentAnalysis, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.analysis, nil
}
func (s *stubProvider) EvaluateAmbiguities(context.Context, *IntentAnalysis) ([]AmbiguityItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.unknowns, nil
}
func (s *stubProvider) GeneratePlanOutline(context.Context, *IntentAnalysis, map[string]string) ([]WorkPackageOutline, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.outline, nil
}

func TestNexusEngine_AnalyzeRequiresConfiguredProvider(t *testing.T) {
	engine := NewNexusEngine(nil)
	_, _, err := engine.Analyze(context.Background(), "Create authentication", "proj-1")
	if !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected ErrIntelligenceUnavailable, got %v", err)
	}
}

func TestNexusEngine_AnalyzeAndClarifyUsesProvider(t *testing.T) {
	provider := &stubProvider{
		analysis: &IntentAnalysis{Intent: "auth", Scope: "project", RiskLevel: "high"},
		unknowns: []AmbiguityItem{{Key: "execution_isolation", Level: AmbiguityImportant, Question: "Isolation?"}},
	}
	engine := NewNexusEngine(provider)

	intent, unknowns, err := engine.Analyze(context.Background(), "Create a full authentication module with database migration", "proj-1")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if intent.RiskLevel != "high" {
		t.Errorf("expected provider risk level, got %s", intent.RiskLevel)
	}
	if len(unknowns) != 1 {
		t.Fatalf("expected provider ambiguity items, got %d", len(unknowns))
	}

	state := &ClarificationState{Unknowns: unknowns}
	engine.ResolveClarification(state, "execution_isolation", "Worktree Isolado")
	if state.StructuredFacts["execution_isolation"] != "Worktree Isolado" {
		t.Errorf("expected fact to be stored, got %v", state.StructuredFacts)
	}
}

func TestNexusEngine_GeneratePlanUsesConfiguredProvider(t *testing.T) {
	expected := []WorkPackageOutline{{Title: "Real provider package", Goal: "ship", Priority: "HIGH"}}
	provider := &stubProvider{outline: expected}
	engine := NewNexusEngine(provider)
	intent := &IntentAnalysis{Intent: "ship", Scope: "project"}

	got, err := engine.GeneratePlan(context.Background(), intent, map[string]string{"platform": "web"})
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}
	if len(got) != 1 || got[0].Title != expected[0].Title {
		t.Fatalf("expected provider outline, got %#v", got)
	}
}

func TestNexusEngine_CompilePromptDoesNotRequireProvider(t *testing.T) {
	engine := NewNexusEngine(nil)
	pkg := WorkPackageOutline{
		Title: "Auth Token Validation", Goal: "Validate Bearer JWT tokens in middleware",
		Priority: "CRITICAL", Role: "implementer",
		Acceptance: []string{"Returns 401 on expired token", "Sets user context on valid token"},
	}
	facts := map[string]string{"auth_scheme": "Bearer JWT with RS256", "token_ttl": "24h"}
	res, err := engine.CompilePrompt(context.Background(), pkg, facts, []string{"verification"})
	if err != nil {
		t.Fatalf("CompilePrompt failed: %v", err)
	}
	if res.PackageTitle != "Auth Token Validation" {
		t.Errorf("unexpected title %s", res.PackageTitle)
	}
	if len(res.AcceptanceGates) != 2 {
		t.Errorf("expected 2 acceptance gates, got %d", len(res.AcceptanceGates))
	}
	if res.EstimatedTokens <= 0 {
		t.Errorf("expected positive token estimate, got %d", res.EstimatedTokens)
	}
}
