package handoff

import (
	"strings"
	"testing"
)

func TestContextEnvelopeRedactionAndFormatting(t *testing.T) {
	sensitiveGoal := "Fix billing logic with API key sk-1234567890abcdef1234567890 and Bearer 1234567890abcdef"
	env := ExtractContextEnvelope("/fake/workspace", "codex", "work", "sess-123", sensitiveGoal)

	if strings.Contains(env.Goal, "sk-1234567890") {
		t.Errorf("API key was NOT redacted from goal: %s", env.Goal)
	}
	if !strings.Contains(env.Goal, "[REDACTED_OPENAI_KEY]") {
		t.Errorf("expected redacted marker in goal, got %s", env.Goal)
	}

	prompt := env.FormatKickoffPrompt()
	if !strings.Contains(prompt, "=== AI Control Context Handoff (from CODEX:work) ===") {
		t.Errorf("kickoff prompt missing header: %s", prompt)
	}
	if !strings.Contains(prompt, "Workspace: /fake/workspace") {
		t.Errorf("kickoff prompt missing workspace: %s", prompt)
	}
}
