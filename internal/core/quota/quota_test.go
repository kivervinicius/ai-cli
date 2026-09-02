package quota

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestQuotaEnginePersistenceAndRendering(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	engine := NewEngine(5 * time.Minute)

	// Test 1: Empty cache returns UNKNOWN, not 100%
	snap, found := engine.GetCachedUsage("codex", "work")
	if found {
		t.Fatalf("expected no cached usage, got found=true")
	}
	if snap.Status != model.UsageUnknown {
		t.Fatalf("expected UNKNOWN status, got %s", snap.Status)
	}

	// Test 2: Progress bar for UNKNOWN
	unknownBar := RenderShortStatus(model.UsageUnknown, nil, 10)
	if !strings.Contains(unknownBar, "UNKNOWN") {
		t.Fatalf("unexpected unknown bar rendering: %s", unknownBar)
	}

	// Test 3: Save and Load Live Snapshot
	remPct := 75.5
	usedPct := 24.5
	liveSnap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{
				Kind:             "5h",
				RemainingPercent: &remPct,
				UsedPercent:      &usedPct,
			},
		},
	}

	if err := engine.SaveUsage(liveSnap); err != nil {
		t.Fatal(err)
	}

	cached, found := engine.GetCachedUsage("codex", "work")
	if !found {
		t.Fatalf("expected cached usage to be found")
	}
	if cached.Status != model.UsageCached {
		t.Fatalf("expected status CACHED when read from cache, got %s", cached.Status)
	}
	if len(cached.Windows) != 1 || *cached.Windows[0].RemainingPercent != 75.5 {
		t.Fatalf("unexpected cached window percent: %+v", cached.Windows)
	}

	// Test 4: Render known progress bar
	knownBar := RenderShortStatus(model.UsageCached, cached.Windows[0].RemainingPercent, 10)
	if !strings.Contains(knownBar, "76%") && !strings.Contains(knownBar, "75%") {
		t.Fatalf("unexpected known bar rendering: %s", knownBar)
	}
}

func TestFetchBatchConcurrency(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	engine := NewEngine(5 * time.Minute)

	profiles := []model.Profile{
		{Provider: "codex", Name: "acc1"},
		{Provider: "codex", Name: "acc2"},
		{Provider: "agy", Name: "acc3"},
	}

	ctx := context.Background()
	results := engine.FetchBatch(ctx, profiles, func(ctx context.Context, p model.Profile) model.UsageSnapshot {
		pct := 50.0
		return model.UsageSnapshot{
			ProviderID: p.Provider,
			ProfileID:  p.Name,
			Status:     model.UsageLive,
			Source:     model.SourceObservation,
			FetchedAt:  time.Now(),
			Windows: []model.UsageWindow{
				{
					Kind:             "5h",
					RemainingPercent: &pct,
				},
			},
		}
	}, 2)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != model.UsageLive {
			t.Errorf("expected LIVE status, got %s", r.Status)
		}
	}
}

func TestQuotaViewBottleneckUnknownWithoutWindowsIsZero(t *testing.T) {
	qv := QuotaView{Status: "UNKNOWN"}
	remaining, kind := qv.Bottleneck()
	if remaining != 0 || kind != "" {
		t.Fatalf("unknown quota without windows must not imply 100%% remaining, got %.1f %q", remaining, kind)
	}
}

func TestLegacyAGYQuotaInvertsConsumedPercent(t *testing.T) {
	if got := legacyAGYRemaining(100); got != 0 {
		t.Fatalf("100%% consumed must be 0%% remaining, got %v", got)
	}
	if got := legacyAGYRemaining(90.15); got < 9.84 || got > 9.86 {
		t.Fatalf("90.15%% consumed must be about 9.85%% remaining, got %v", got)
	}
	if got := legacyAGYRemaining(0); got != 100 {
		t.Fatalf("0%% consumed must be 100%% remaining, got %v", got)
	}
}
