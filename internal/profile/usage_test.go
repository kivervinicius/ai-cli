package profile

import (
	"os"
	"path/filepath"
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
	if _, err := Create("agy", "work"); err != nil {
		t.Fatal(err)
	}
	home, _ := Home("agy", "work")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"work@example.test"}`), 0600); err != nil {
		t.Fatal(err)
	}

	remaining := 66.0
	used := 34.0
	eng := quota.NewEngine(time.Minute)
	scope, err := AccountScope("agy", "work", "work@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveUsageForScope(scope, model.UsageSnapshot{
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

func TestGetUsageSnapshotKeepsCodexLiveRolloutWithoutPrefixedScope(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	if _, err := Create("codex", "omega"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("codex", "omega")
	if err != nil {
		t.Fatal(err)
	}
	sessDir := filepath.Join(home, ".codex", "sessions", "2026", "09", "10")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	auth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImtpdmVyQG9tZWdhc2lzdGVtYXMubmV0LmJyIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicGx1cyJ9fQ.sig"}}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	rollout := `{"type":"session_meta","payload":{"session_id":"s1"}}
{"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":10.0,"window_minutes":300,"resets_at":1788556157},"secondary":{"used_percent":20.0,"window_minutes":10080,"resets_at":1788748886}}}}
`
	if err := os.WriteFile(filepath.Join(sessDir, "rollout-live.jsonl"), []byte(rollout), 0o600); err != nil {
		t.Fatal(err)
	}

	snap := GetUsageSnapshot("codex", "omega")
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE (adapter rollout must not be discarded for missing AccountScope)", snap.Status)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("windows=%d want 2", len(snap.Windows))
	}
	if !snap.AccountScope.Verifiable() {
		t.Fatal("live Codex snapshot should be stamped with the profile account scope")
	}
}

func TestSnapshotBelongsToProfileRejectsForeignScope(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	if _, err := Create("codex", "omega"); err != nil {
		t.Fatal(err)
	}
	scope, err := AccountScope("codex", "omega", "kiver@omegasistemas.net.br")
	if err != nil {
		t.Fatal(err)
	}
	foreign := scope
	foreign.AccountID = "other-account"
	snap := model.UsageSnapshot{
		ProviderID:   "codex",
		ProfileID:    "omega",
		Account:      "kiver@omegasistemas.net.br",
		AccountScope: foreign,
		Status:       model.UsageLive,
	}
	if snapshotBelongsToProfile(snap, "codex", "omega", "kiver@omegasistemas.net.br") {
		t.Fatal("foreign verifiable scope must be rejected")
	}
	snap.AccountScope = model.AccountScope{}
	if !snapshotBelongsToProfile(snap, "codex", "omega", "kiver@omegasistemas.net.br") {
		t.Fatal("live adapter snapshot without AccountScope must be accepted when email matches")
	}
}
