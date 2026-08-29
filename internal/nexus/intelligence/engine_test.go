package intelligence

import (
	"context"
	"testing"
)

func TestNexusEngine_AnalyzeAndClarify(t *testing.T) {
	engine := NewNexusEngine(nil)

	intent, unknowns, err := engine.Analyze(context.Background(), "Create a full authentication module with database migration", "proj-1")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if intent.RiskLevel != "high" {
		t.Errorf("expected high risk level for database migration intent, got %s", intent.RiskLevel)
	}

	if len(unknowns) == 0 {
		t.Fatalf("expected ambiguity items to be identified, got 0")
	}

	state := &ClarificationState{
		Unknowns: unknowns,
	}

	// Resolve an ambiguity
	engine.ResolveClarification(state, "execution_isolation", "Worktree Isolado")
	if state.StructuredFacts["execution_isolation"] != "Worktree Isolado" {
		t.Errorf("expected fact to be stored, got %v", state.StructuredFacts)
	}
}

func TestNexusEngine_CompilePrompt(t *testing.T) {
	engine := NewNexusEngine(nil)

	pkg := WorkPackageOutline{
		Title:      "Auth Token Validation",
		Goal:       "Validate Bearer JWT tokens in middleware",
		Priority:   "CRITICAL",
		Role:       "implementer",
		Acceptance: []string{"Returns 401 on expired token", "Sets user context on valid token"},
	}

	facts := map[string]string{
		"auth_scheme": "Bearer JWT with RS256",
		"token_ttl":   "24h",
	}

	skills := []string{"skill-security-audit", "skill-verification"}

	res, err := engine.CompilePrompt(context.Background(), pkg, facts, skills)
	if err != nil {
		t.Fatalf("CompilePrompt failed: %v", err)
	}

	if res.PackageTitle != "Auth Token Validation" {
		t.Errorf("expected package title in result, got %s", res.PackageTitle)
	}

	if len(res.AcceptanceGates) != 2 {
		t.Errorf("expected 2 acceptance gates, got %d", len(res.AcceptanceGates))
	}

	if res.EstimatedTokens <= 0 {
		t.Errorf("expected positive token estimate, got %d", res.EstimatedTokens)
	}
}
