package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

func TestSmartAccountSelector(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)

	ctx := context.Background()

	// Scenario 1: Profile A has 20%, Profile B has 85% -> B must be selected
	pctA := 20.0
	pctB := 85.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-a",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &pctA}},
	})
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-b",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &pctB}},
	})

	candidates := []model.Profile{
		{Provider: "codex", Name: "acc-a"},
		{Provider: "codex", Name: "acc-b"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-a": {Authenticated: true, Health: model.HealthHealthy},
		"acc-b": {Authenticated: true, Health: model.HealthHealthy},
	}

	res1, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res1.SelectedProfile.Name != "acc-b" {
		t.Fatalf("expected acc-b (85%%) to be selected, got %s", res1.SelectedProfile.Name)
	}

	// Scenario 2: Profile B becomes RATE_LIMITED -> acc-a (20%) must be selected
	cdTracker.RecordRateLimit("codex", "acc-b", 30*time.Minute, nil, "HTTP 429")
	res2, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.SelectedProfile.Name != "acc-a" {
		t.Fatalf("expected acc-a to be selected when acc-b is rate limited, got %s", res2.SelectedProfile.Name)
	}

	// Scenario 3: UNKNOWN healthy vs UNKNOWN rate-limited -> selects UNKNOWN healthy
	cdTracker.Clear("codex", "acc-b")
	cdTracker.RecordRateLimit("codex", "acc-a", 15*time.Minute, nil, "HTTP 429")
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-b",
		Status:     model.UsageUnknown, // UNKNOWN
		Windows:    nil,
	})

	res3, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res3.SelectedProfile.Name != "acc-b" {
		t.Fatalf("expected acc-b to be selected, got %s", res3.SelectedProfile.Name)
	}

	// Scenario 4: All accounts rate limited -> returns error with reasons
	cdTracker.RecordRateLimit("codex", "acc-b", 15*time.Minute, nil, "HTTP 429")
	res4, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err == nil {
		t.Fatalf("expected error when all accounts are rate limited, got selected=%+v", res4)
	}
	if !strings.Contains(err.Error(), "no usable codex profiles") {
		t.Fatalf("expected descriptive error message, got %v", err)
	}

	// Scenario 5: Explain Selection
	explain := selector.ExplainSelection(ctx, "codex", "/tmp", candidates, accounts)
	if !strings.Contains(explain, "Evaluation of all candidate profiles") {
		t.Fatalf("unexpected explain output: %s", explain)
	}
}

func TestMultiQuotaBottleneckSelection(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	// acc-a is default in config
	cfg.Defaults = map[string]string{
		"agy": "acc-a",
	}

	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)
	ctx := context.Background()

	// Profile A (default): Google 5h is 92%, but Claude 5h is 10% (bottleneck)
	p5hA := 92.0
	pClaudeA := 10.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-a",
		Status:     model.UsageLive,
		Windows: []model.UsageWindow{
			{Kind: "5h", RemainingPercent: &p5hA},
			{Kind: "claude_5h", RemainingPercent: &pClaudeA},
		},
	})

	// Profile B: Google 5h is 90%, Claude 5h is 90%
	p5hB := 90.0
	pClaudeB := 90.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-b",
		Status:     model.UsageLive,
		Windows: []model.UsageWindow{
			{Kind: "5h", RemainingPercent: &p5hB},
			{Kind: "claude_5h", RemainingPercent: &pClaudeB},
		},
	})

	candidates := []model.Profile{
		{Provider: "agy", Name: "acc-a"},
		{Provider: "agy", Name: "acc-b"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-a": {Authenticated: true, Health: model.HealthHealthy},
		"acc-b": {Authenticated: true, Health: model.HealthHealthy},
	}

	res, err := selector.SelectBestProfile(ctx, "agy", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedProfile.Name != "acc-b" {
		t.Fatalf("expected acc-b to be selected due to higher overall token availability (claude bottleneck on acc-a), got %s", res.SelectedProfile.Name)
	}
}

func TestDefaultProfileTieBreaker(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	cfg.Defaults = map[string]string{
		"agy": "acc-default",
	}

	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)
	ctx := context.Background()

	// Both have 100% capacity
	p100 := 100.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-default",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &p100}},
	})
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-other",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &p100}},
	})

	candidates := []model.Profile{
		{Provider: "agy", Name: "acc-other"},
		{Provider: "agy", Name: "acc-default"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-other":   {Authenticated: true, Health: model.HealthHealthy},
		"acc-default": {Authenticated: true, Health: model.HealthHealthy},
	}

	res, err := selector.SelectBestProfile(ctx, "agy", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedProfile.Name != "acc-default" {
		t.Fatalf("expected acc-default to win tie-break on equal 100%% capacity, got %s", res.SelectedProfile.Name)
	}
}
