package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
)

const (
	DefaultTTL = 5 * time.Minute
)

var (
	cacheMu sync.RWMutex
)

// Engine manages reading, refreshing, caching, and rendering quotas honestly.
type Engine struct {
	ttl time.Duration
}

// NewEngine creates a new Quota Engine instance.
func NewEngine(ttl time.Duration) *Engine {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Engine{ttl: ttl}
}

// GetCachedUsage returns the cached snapshot for a profile without triggering external requests.
func (e *Engine) GetCachedUsage(provider, profileName string) (model.UsageSnapshot, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	root, err := config.ProfileRoot(provider, profileName)
	if err != nil {
		return model.UsageSnapshot{
			ProviderID: provider,
			ProfileID:  profileName,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		}, false
	}

	quotaFile := filepath.Join(root, "usage.json")
	data, err := os.ReadFile(quotaFile)
	if err != nil {
		// Fallback to legacy quota.json if available
		legacyFile := filepath.Join(root, "quota.json")
		data, err = os.ReadFile(legacyFile)
		if err != nil {
			return model.UsageSnapshot{
				ProviderID: provider,
				ProfileID:  profileName,
				Status:     model.UsageUnknown,
				Source:     model.SourceNone,
			}, false
		}
	}

	var snap model.UsageSnapshot
	_ = json.Unmarshal(data, &snap)

	if snap.Status == "" || len(snap.Windows) == 0 {
		var leg struct {
			ModelName string `json:"model_name"`
			Plan      string `json:"plan"`
			FiveHour  struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"five_hour"`
			Weekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"weekly"`
			ClaudeFiveHour struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"claude_five_hour"`
			ClaudeWeekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"claude_weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetTime != "" || leg.Weekly.ResetTime != "" || leg.FiveHour.ResetsIn != "" || leg.Weekly.ResetsIn != "" || leg.ClaudeFiveHour.PercentLeft > 0 || leg.ClaudeWeekly.PercentLeft > 0) {
			p5h := leg.FiveHour.PercentLeft
			u5h := 100.0 - p5h
			pWk := leg.Weekly.PercentLeft
			uWk := 100.0 - pWk
			r5h := leg.FiveHour.ResetTime
			if r5h == "" {
				r5h = leg.FiveHour.ResetsIn
			}
			rWk := leg.Weekly.ResetTime
			if rWk == "" {
				rWk = leg.Weekly.ResetsIn
			}

			windows := []model.UsageWindow{
				{
					Kind:             "5h",
					Group:            "gemini",
					RemainingPercent: &p5h,
					UsedPercent:      &u5h,
					ResetDescription: r5h,
				},
				{
					Kind:             "weekly",
					Group:            "gemini",
					RemainingPercent: &pWk,
					UsedPercent:      &uWk,
					ResetDescription: rWk,
				},
			}

			if leg.ClaudeFiveHour.PercentLeft > 0 || leg.ClaudeFiveHour.ResetsIn != "" || leg.ClaudeFiveHour.ResetTime != "" {
				pC5h := leg.ClaudeFiveHour.PercentLeft
				uC5h := 100.0 - pC5h
				rC5h := leg.ClaudeFiveHour.ResetTime
				if rC5h == "" {
					rC5h = leg.ClaudeFiveHour.ResetsIn
				}
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_5h",
					Group:            "claude_gpt",
					RemainingPercent: &pC5h,
					UsedPercent:      &uC5h,
					ResetDescription: rC5h,
				})
			}

			if leg.ClaudeWeekly.PercentLeft > 0 || leg.ClaudeWeekly.ResetsIn != "" || leg.ClaudeWeekly.ResetTime != "" {
				pCWk := leg.ClaudeWeekly.PercentLeft
				uCWk := 100.0 - pCWk
				rCWk := leg.ClaudeWeekly.ResetTime
				if rCWk == "" {
					rCWk = leg.ClaudeWeekly.ResetsIn
				}
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_weekly",
					Group:            "claude_gpt",
					RemainingPercent: &pCWk,
					UsedPercent:      &uCWk,
					ResetDescription: rCWk,
				})
			}

			snap = model.UsageSnapshot{
				ProviderID: provider,
				ProfileID:  profileName,
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  time.Now(),
				Windows:    windows,
			}
			return snap, true
		}
		return model.UsageSnapshot{
			ProviderID: provider,
			ProfileID:  profileName,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		}, false
	}

	// Mark status as CACHED if it was originally LIVE
	if snap.Status == model.UsageLive {
		snap.Status = model.UsageCached
	}

	return snap, true
}

// SaveUsage persists a usage snapshot to the profile directory.
func (e *Engine) SaveUsage(snap model.UsageSnapshot) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	root, err := config.ProfileRoot(snap.ProviderID, snap.ProfileID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	quotaFile := filepath.Join(root, "usage.json")
	tempFile := filepath.Join(root, fmt.Sprintf("usage.tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(tempFile, append(data, '\n'), 0600); err != nil {
		return err
	}

	return os.Rename(tempFile, quotaFile)
}

// FormatFreshness returns human-readable age of a snapshot (e.g. "updated 12s ago").
func FormatFreshness(fetchedAt time.Time) string {
	if fetchedAt.IsZero() {
		return "never updated"
	}
	d := time.Since(fetchedAt)
	if d < 10*time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}

// RenderProgressBar builds an honest visual progress bar.
func RenderProgressBar(status model.UsageStatus, remainingPercent *float64, width int) string {
	if width < 6 {
		width = 10
	}

	switch status {
	case model.UsageLive, model.UsageCached, model.UsageEstimated:
		if remainingPercent == nil {
			return formatProgressLabel("UNKNOWN", width)
		}
		pct := *remainingPercent
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		fillCount := int((pct / 100.0) * float64(width))
		if fillCount > width {
			fillCount = width
		}
		emptyCount := width - fillCount

		// On Windows console (cmd.exe, PowerShell, legacy OEM code pages CP437/CP850/CP1252),
		// UTF-8 block characters (█, ░) frequently render as '???' or mojibake.
		// We use universal ASCII characters (# and -) on Windows to ensure 100% clean rendering.
		if runtime.GOOS == "windows" {
			return "[" + strings.Repeat("#", fillCount) + strings.Repeat("-", emptyCount) + "]"
		}
		return "[" + strings.Repeat("█", fillCount) + strings.Repeat("░", emptyCount) + "]"

	case model.UsageRateLimited:
		return formatProgressLabel("LIMITED", width)

	case model.UsageUnsupported:
		return formatProgressLabel("UNSUPPORT", width)

	case model.UsageError:
		return formatProgressLabel("ERROR", width)

	default:
		return formatProgressLabel("UNKNOWN", width)
	}
}

func formatProgressLabel(label string, width int) string {
	if len(label) >= width {
		return "[" + label[:width] + "]"
	}
	pad := width - len(label)
	left := pad / 2
	right := pad - left
	return "[" + strings.Repeat(" ", left) + label + strings.Repeat(" ", right) + "]"
}

// RenderShortStatus returns a compact 1-line representation for tables and status bars.
func RenderShortStatus(status model.UsageStatus, remainingPercent *float64, width int) string {
	bar := RenderProgressBar(status, remainingPercent, width)
	if (status == model.UsageLive || status == model.UsageCached) && remainingPercent != nil {
		return fmt.Sprintf("%s %3.0f%% (%s)", bar, *remainingPercent, string(status))
	}
	return fmt.Sprintf("%s (%s)", bar, string(status))
}

// BatchFetch retrieves usage snapshots for multiple profiles with controlled concurrency.
func (e *Engine) BatchFetch(ctx context.Context, profiles []model.Profile, fetcher func(ctx context.Context, p model.Profile) model.UsageSnapshot, maxWorkers int) map[string]model.UsageSnapshot {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	results := make(map[string]model.UsageSnapshot)
	var mu sync.Mutex

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, p := range profiles {
		wg.Add(1)
		go func(prof model.Profile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			snap := fetcher(ctx, prof)
			_ = e.SaveUsage(snap)

			mu.Lock()
			results[prof.Provider+":"+prof.Name] = snap
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	return results
}

// FetchBatch is an alias for BatchFetch.
func (e *Engine) FetchBatch(ctx context.Context, profiles []model.Profile, fetcher func(ctx context.Context, p model.Profile) model.UsageSnapshot, maxWorkers int) map[string]model.UsageSnapshot {
	return e.BatchFetch(ctx, profiles, fetcher, maxWorkers)
}
