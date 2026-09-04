package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

func TestExtractJSONObjectIgnoresTrailingProseWithBraces(t *testing.T) {
	payload, err := extractJSONObject(`{"intent":"quero ideias","scope":"project","risk_level":"low","identified_goals":["ideas"],"constraints":[],"assumptions":[]}
OK. Prefer {option A} over {option B}.`)
	if err != nil {
		t.Fatal(err)
	}
	var result IntentAnalysis
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Intent != "quero ideias" {
		t.Fatalf("unexpected intent %q", result.Intent)
	}
}

func TestExtractJSONObjectRejectsNonObject(t *testing.T) {
	if _, err := extractJSONObject(`OK no json here`); err == nil {
		t.Fatal("expected error")
	}
}


func TestHeadlessPromptArgsMatchSupportedProviderCLIs(t *testing.T) {
	cases := []struct {
		provider string
		want0    string
	}{
		{"claude", "-p"}, {"agy", "-p"}, {"gemini", "-p"}, {"cursor", "-p"}, {"opencode", "run"}, {"codex", "exec"},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			args, err := HeadlessPromptArgs(tc.provider, "hello")
			if err != nil {
				t.Fatal(err)
			}
			if len(args) != 2 || args[0] != tc.want0 || args[1] != "hello" {
				t.Fatalf("unexpected args %#v", args)
			}
		})
	}
}

func TestCLIProviderPlanIncludesProjectContext(t *testing.T) {
	var prompt string
	p := NewCLIProvider("claude", "work", true, func(_ context.Context, input string) (string, error) {
		prompt = input
		return `{"packages":[{"title":"Build","goal":"ship","priority":"NORMAL","dependencies":[],"role":"implementer","skills":[],"acceptance":["done"]}]}`, nil
	})
	_, err := p.GeneratePlanOutline(context.Background(), &IntentAnalysis{Intent: "ship"}, nil, map[string]any{"project_context": map[string]any{"head": "abc123"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "abc123") || !strings.Contains(prompt, "project_context") {
		t.Fatalf("CLI plan prompt missing bounded project context: %s", prompt)
	}
}
