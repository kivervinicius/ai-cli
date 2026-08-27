package profile

import (
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestGetQuotaDetails(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tempData)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Without cached data, quota must report UNKNOWN, NOT 100%
	qAgy := GetQuotaDetails("agy", "test-profile-1", "Google AI Pro", "test@gmail.com")
	if qAgy.Status != string(model.UsageUnknown) {
		t.Fatalf("expected UNKNOWN status when quota is missing, got: %+v", qAgy)
	}

	qCodex := GetQuotaDetails("codex", "test-profile-2", "ChatGPT Plus", "test@openai.com")
	if qCodex.Status != string(model.UsageUnknown) {
		t.Fatalf("expected UNKNOWN status when quota is missing, got: %+v", qCodex)
	}

	// Bar rendering test
	b := RenderBar(70.0, 20)
	if len(b) == 0 {
		t.Fatal("empty rendered bar")
	}
}
