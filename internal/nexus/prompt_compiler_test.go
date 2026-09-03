package nexus

import (
	"strings"
	"testing"
)

func TestCompileAgentPromptWithoutSkills(t *testing.T) {
	prompt := "Run tests and refactor"
	compiled, err := CompileAgentPrompt(prompt, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled.CompiledPrompt != prompt {
		t.Fatalf("expected untouched prompt %q, got %q", prompt, compiled.CompiledPrompt)
	}
	if len(compiled.ValidatedSkills) != 0 {
		t.Fatalf("expected no validated skills, got %v", compiled.ValidatedSkills)
	}
	if compiled.PromptHash == "" {
		t.Fatal("expected prompt hash to be calculated")
	}
}

func TestCompileAgentPromptWithValidatedSkills(t *testing.T) {
	client := &MaestroClient{
		status: MaestroStatus{
			Available: true,
			Mode:      MaestroAssist,
			Capabilities: &MaestroCapability{
				Version: "1.0.0",
				Skills: []MaestroSkillDesc{
					{ID: "skill-security-review", Name: "Security Review"},
					{ID: "skill-repo-health", Name: "Repo Health"},
				},
			},
		},
	}

	userPrompt := "Revise a segurança deste endpoint"
	compiled, err := CompileAgentPrompt(userPrompt, []string{"skill-security-review"}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(compiled.CompiledPrompt, "Nexus execution context") {
		t.Fatalf("expected context envelope, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, "Scope: next prompt only") {
		t.Fatalf("expected scope, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, "skill-security-review") {
		t.Fatalf("expected validated skill in compiled prompt, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, userPrompt) {
		t.Fatalf("expected user prompt in compiled prompt, got: %s", compiled.CompiledPrompt)
	}
}

func TestCompileAgentPromptRejectsDegradedMaestro(t *testing.T) {
	client := &MaestroClient{
		status: MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     "maestro offline",
		},
	}

	_, err := CompileAgentPrompt("test prompt", []string{"skill-security-review"}, client)
	if err == nil {
		t.Fatal("expected failure when maestro is offline and skills are requested")
	}
	if !strings.Contains(err.Error(), "MAESTRO_DEGRADED") {
		t.Fatalf("expected MAESTRO_DEGRADED error, got: %v", err)
	}
}
