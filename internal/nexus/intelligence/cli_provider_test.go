package intelligence

import (
	"context"
	"errors"
	"testing"
)

func TestCLIProviderUnavailableWhenBackendIsNotCapabilityValidated(t *testing.T) {
	p := NewCLIProvider("claude", "work", false, func(context.Context, string) (string, error) {
		return "", nil
	})
	if p.Available(context.Background()) {
		t.Fatal("CLI intelligence must be unavailable until the Nexus adapter validates headless + submit_prompt")
	}
	_, err := p.AnalyzeIntent(context.Background(), "ship auth", nil)
	if !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected ErrIntelligenceUnavailable, got %v", err)
	}
}

func TestCLIProviderAnalyzeParsesJSONEnvelope(t *testing.T) {
	var gotPrompt string
	p := NewCLIProvider("claude", "work", true, func(_ context.Context, prompt string) (string, error) {
		gotPrompt = prompt
		return "prefix\n```json\n{\"intent\":\"ship auth\",\"scope\":\"project\",\"risk_level\":\"high\",\"identified_goals\":[\"auth\"],\"constraints\":[],\"assumptions\":[]}\n```\nsuffix", nil
	})
	got, err := p.AnalyzeIntent(context.Background(), "ship auth", nil)
	if err != nil {
		t.Fatalf("AnalyzeIntent: %v", err)
	}
	if got.Intent != "ship auth" || got.RiskLevel != "high" {
		t.Fatalf("unexpected analysis: %+v", got)
	}
	if gotPrompt == "" {
		t.Fatal("expected a compiled CLI prompt")
	}
}
