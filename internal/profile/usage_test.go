package profile

import (
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
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

func TestGetUsageSnapshotPreservesStaleWindowsAsEstimatedAfterRefreshFailure(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tempData)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	remaining := 66.0
	used := 34.0
	eng := quota.NewEngine(time.Minute)
	if err := eng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now().Add(-time.Hour),
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "gemini",
			RemainingPercent: &remaining,
			UsedPercent:      &used,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	snap := GetUsageSnapshot("agy", "work")
	if snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s, want ESTIMATED for stale cache fallback", snap.Status)
	}
	if len(snap.Windows) != 1 {
		t.Fatalf("expected last-known windows to be preserved, got %+v", snap.Windows)
	}
	if snap.Error == "" {
		t.Fatal("expected diagnostic explaining the degraded snapshot")
	}
}
