package profile

import (
	"testing"
)

func TestGetQuotaDetails(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", tempData)
	t.Setenv("AI_MANAGER_CONFIG_DIR", t.TempDir())

	qAgy := GetQuotaDetails("agy", "test-profile-1", "Google AI Pro", "test@gmail.com")
	if qAgy.FiveHour.PercentLeft != 100.0 || qAgy.Weekly.PercentLeft != 100.0 {
		t.Fatalf("unexpected AGY quota: %+v", qAgy)
	}

	qCodex := GetQuotaDetails("codex", "test-profile-2", "ChatGPT Plus", "test@openai.com")
	if qCodex.FiveHour.PercentLeft != 100.0 || qCodex.Weekly.PercentLeft != 100.0 {
		t.Fatalf("unexpected Codex quota: %+v", qCodex)
	}

	// Bar rendering test
	b := RenderBar(70.0, 20)
	if len(b) == 0 {
		t.Fatal("empty rendered bar")
	}
}
