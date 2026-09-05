package profile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

type LimitWindow struct {
	PercentLeft float64 `json:"percent_left"`
	ResetsIn    string  `json:"resets_in"`
	ResetTime   string  `json:"reset_time"`
	ProgressBar string  `json:"progress_bar"`
	Status      string  `json:"status,omitempty"`
}

type QuotaDetails struct {
	Provider    string      `json:"provider"`
	ProfileName string      `json:"profile_name"`
	Account     string      `json:"account"`
	Plan        string      `json:"plan"`
	Status      string      `json:"status"`
	ModelName   string      `json:"model_name"`
	FiveHour    LimitWindow `json:"five_hour"`
	Weekly      LimitWindow `json:"weekly"`
	ClaudeFiveH LimitWindow `json:"claude_five_hour,omitempty"`
	ClaudeWeek  LimitWindow `json:"claude_weekly,omitempty"`
}

// GetUsageSnapshot returns a point-in-time usage snapshot for a provider and profile.
// Cached windows are returned immediately so interactive commands such as
// `nexus usage` never block on a live provider CLI.
func GetUsageSnapshot(providerName, name string) model.UsageSnapshot {
	return loadUsageSnapshot(providerName, name, false)
}

// RefreshUsageSnapshot forces a provider fetch and persists the result when live.
func RefreshUsageSnapshot(providerName, name string) model.UsageSnapshot {
	return loadUsageSnapshot(providerName, name, true)
}

func loadUsageSnapshot(providerName, name string, refresh bool) model.UsageSnapshot {
	qEng := quota.NewEngine(5 * time.Minute)
	snap, found := qEng.GetCachedUsage(providerName, name)
	// Codex rollouts are local filesystem reads (not a blocking CLI). Always
	// prefer the adapter so stale quota.json with phantom fields cannot hide
	// the live primary/secondary used_percent from recent sessions.
	useCache := !refresh && found && len(snap.Windows) > 0 && snap.Status != model.UsageUnknown
	if useCache && providerName == "codex" {
		useCache = false
	}
	if useCache {
		return snap
	}

	ctx := context.Background()
	p := model.Profile{Provider: providerName, Name: name}
	switch providerName {
	case "codex":
		snap = codex.New().GetUsage(ctx, p)
	case "agy":
		if refresh {
			snap = agy.New().RefreshUsage(ctx, p)
		} else {
			snap = agy.New().GetUsage(ctx, p)
		}
	case "claude":
		snap = claude.New().GetUsage(ctx, p)
	case "opencode":
		snap = opencode.New().GetUsage(ctx, p)
	case "gemini":
		snap = gemini.New().GetUsage(ctx, p)
	default:
		snap = model.UsageSnapshot{
			ProviderID: providerName,
			ProfileID:  name,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
			FetchedAt:  time.Now(),
		}
	}

	// Reject stale snapshots from adapters: a quota file older than the
	// trust window is not evidence of live capacity. Downgrading to
	// UNKNOWN prevents the scheduler and usage tables from displaying
	// phantom 100% remaining data from disk caches written days ago.
	if snap.Status != model.UsageUnknown && snap.Status != model.UsageError && len(snap.Windows) > 0 {
		if !qEng.Trustworthy(snap) {
			snap.Status = model.UsageUnknown
			snap.Source = model.SourceNone
			snap.Windows = nil
			return snap
		}
		// Persist only trustworthy data so the scheduler and other
		// consumers read current quota instead of stale cache files.
		eng := quota.NewEngine(5 * time.Minute)
		_ = eng.SaveUsage(snap)
	}

	return snap
}

// GetQuotaDetails returns usage and quota metrics without fabricating 100% data.
func GetQuotaDetails(providerName, name, plan, email string) QuotaDetails {
	snap := GetUsageSnapshot(providerName, name)

	q := QuotaDetails{
		Provider:    providerName,
		ProfileName: name,
		Account:     email,
		Plan:        plan,
		Status:      string(snap.Status),
		ModelName:   snap.ModelName,
	}

	if q.ModelName == "" {
		switch providerName {
		case "codex":
			q.ModelName = "gpt-5.6-sol"
		case "agy":
			q.ModelName = "Gemini 2.5 Flash / Pro"
		case "claude":
			q.ModelName = "Claude 3.7 Sonnet"
		case "opencode":
			q.ModelName = "OpenCode Provider"
		case "gemini":
			q.ModelName = "Gemini Pro"
		}
	}

	// Map windows if present
	for _, w := range snap.Windows {
		var pct float64
		if w.RemainingPercent != nil {
			pct = *w.RemainingPercent
		}
		bar := quota.RenderProgressBar(snap.Status, w.RemainingPercent, 10)
		if snap.Status == model.UsageLive || snap.Status == model.UsageCached {
			bar = fmt.Sprintf("%s %2.0f%%", bar, pct)
		}
		lw := LimitWindow{
			PercentLeft: pct,
			ResetTime:   w.ResetDescription,
			ResetsIn:    w.ResetDescription,
			ProgressBar: bar,
			Status:      string(snap.Status),
		}
		switch w.Kind {
		case "5h", "daily":
			q.FiveHour = lw
		case "weekly":
			q.Weekly = lw
		case "claude_5h", "claude_five_hour":
			q.ClaudeFiveH = lw
		case "claude_weekly":
			q.ClaudeWeek = lw
		}
	}

	if len(snap.Windows) == 0 {
		q.FiveHour = LimitWindow{
			ProgressBar: quota.RenderProgressBar(snap.Status, nil, 10),
			Status:      string(snap.Status),
			ResetTime:   "Status: " + string(snap.Status),
		}
		q.Weekly = LimitWindow{
			ProgressBar: quota.RenderProgressBar(snap.Status, nil, 10),
			Status:      string(snap.Status),
			ResetTime:   "Status: " + string(snap.Status),
		}
	}

	return q
}

// GetQuotaView returns the omnibus QuotaView for any consumer (TUI, Web, CLI, Scheduler).
// This is the preferred entry point for quota display and scoring.
func GetQuotaView(providerName, name, plan, email string) quota.QuotaView {
	snap := GetUsageSnapshot(providerName, name)
	// A cached quota can outlive an account switch. Do not show or score data
	// whose recorded identity differs from the profile's authenticated email.
	if snapshotAccount := strings.TrimSpace(snap.Account); snapshotAccount != "" && strings.TrimSpace(email) != "" && !strings.EqualFold(snapshotAccount, strings.TrimSpace(email)) {
		snap.Status = model.UsageUnknown
		snap.Source = model.SourceNone
		snap.Windows = nil
		snap.Error = fmt.Sprintf("quota belongs to %s, profile is authenticated as %s", snapshotAccount, strings.TrimSpace(email))
	}

	// Stale local files must not score as fresh CACHED (e.g. August quota.json).
	qEng := quota.NewEngine(quota.DefaultTTL)
	if len(snap.Windows) > 0 && snap.Status != model.UsageUnknown && !qEng.Trustworthy(snap) {
		if snap.Status == model.UsageCached || snap.Status == model.UsageLive {
			snap.Status = model.UsageEstimated
		}
	}

	// Apply default model names when the adapter didn't set one.
	if snap.ModelName == "" {
		snap.ModelName = defaultModelName(providerName)
	}

	return quota.BuildQuotaView(snap, email, plan)
}

// defaultModelName returns a fallback model name for providers that don't set one.
func defaultModelName(provider string) string {
	switch provider {
	case "codex":
		return "gpt-5.6-sol"
	case "agy":
		return "Gemini 2.5 Flash / Pro"
	case "claude":
		return "Claude 3.7 Sonnet"
	case "opencode":
		return "OpenCode Provider"
	case "gemini":
		return "Gemini Pro"
	default:
		return provider
	}
}

// RenderBar generates a clean progress bar.
func RenderBar(percent float64, width int) string {
	p := percent
	return quota.RenderProgressBar(model.UsageLive, &p, width)
}

func RenderShortBar(percent float64) string {
	p := percent
	return quota.RenderShortStatus(model.UsageLive, &p, 10)
}
