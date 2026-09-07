package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
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
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("loadUsageSnapshot: starting", "provider", providerName, "profile", name, "refresh", refresh)
	}

	qEng := quota.NewEngine(5 * time.Minute)
	lastKnown, hasLastKnown := qEng.GetLastKnownUsage(providerName, name)
	if debug {
		slog.Debug("loadUsageSnapshot: GetLastKnownUsage", "provider", providerName, "profile", name, "hasLastKnown", hasLastKnown, "lastKnownStatus", lastKnown.Status, "lastKnownFetchedAt", lastKnown.FetchedAt, "lastKnownWindows", len(lastKnown.Windows))
	}

	snap, found := qEng.GetCachedUsage(providerName, name)
	if debug {
		slog.Debug("loadUsageSnapshot: GetCachedUsage", "provider", providerName, "profile", name, "found", found, "snapStatus", snap.Status, "snapFetchedAt", snap.FetchedAt, "snapWindows", len(snap.Windows), "snapAccount", snap.Account)
	}

	// AGY profiles are per-Google-account. Reject cached snapshots whose
	// recorded account does not match the profile's authenticated email.
	// This prevents stale cross-profile cache from being served.
	if providerName == "agy" && found && snap.Account != "" {
		authEmail := resolveAGYAuthenticatedEmail(name)
		if debug {
			slog.Debug("loadUsageSnapshot: AGY account check", "provider", providerName, "profile", name, "cacheAccount", snap.Account, "authEmail", authEmail)
		}
		if authEmail != "" && !strings.EqualFold(authEmail, snap.Account) {
			if debug {
				slog.Debug("loadUsageSnapshot: AGY account mismatch, rejecting cache", "provider", providerName, "profile", name, "cacheAccount", snap.Account, "authEmail", authEmail)
			}
			snap = model.UsageSnapshot{
				ProviderID: providerName,
				ProfileID:  name,
				Status:     model.UsageUnknown,
				Source:     model.SourceNone,
			}
			found = false
		}
	}

	// Codex rollouts are local filesystem reads (not a blocking CLI). Always
	// prefer the adapter so stale quota.json with phantom fields cannot hide
	// the live primary/secondary used_percent from recent sessions.
	useCache := !refresh && found && len(snap.Windows) > 0 && snap.Status != model.UsageUnknown && qEng.Trustworthy(snap)
	if useCache && providerName == "codex" {
		useCache = false
	}
	if debug {
		slog.Debug("loadUsageSnapshot: cache decision", "provider", providerName, "profile", name, "useCache", useCache, "trustworthy", qEng.Trustworthy(snap))
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

	if debug {
		slog.Debug("loadUsageSnapshot: after adapter", "provider", providerName, "profile", name, "status", snap.Status, "source", snap.Source, "fetchedAt", snap.FetchedAt, "windows", len(snap.Windows), "account", snap.Account)
	}

	// Reject stale snapshots from adapters: a quota file older than the
	// trust window is not evidence of live capacity. Keep processing below so
	// the last-known fallback can preserve context without treating it as live.
	if snap.Status != model.UsageUnknown && snap.Status != model.UsageError && len(snap.Windows) > 0 {
		if !qEng.Trustworthy(snap) {
			if debug {
				slog.Debug("loadUsageSnapshot: rejecting stale adapter snapshot", "provider", providerName, "profile", name, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
			}
			snap.Status = model.UsageUnknown
			snap.Source = model.SourceNone
			snap.Windows = nil
		} else {
			// Persist only trustworthy data so the scheduler and other
			// consumers read current quota instead of stale cache files.
			eng := quota.NewEngine(5 * time.Minute)
			_ = eng.SaveUsage(snap)
			if debug {
				slog.Debug("loadUsageSnapshot: returning trustworthy adapter snapshot", "provider", providerName, "profile", name, "status", snap.Status)
			}
			return snap
		}
	}

	// A failed refresh must not destroy the last successful observation. Keep
	// it visible as ESTIMATED, with an explicit diagnostic, while ensuring it
	// cannot pass the normal freshness check or masquerade as live quota.
	if hasLastKnown && len(lastKnown.Windows) > 0 {
		lastKnown.ProviderID = providerName
		lastKnown.ProfileID = name
		lastKnown.Status = model.UsageEstimated
		lastKnown.Error = fmt.Sprintf("live usage refresh failed; showing last known observation from %s", quota.FormatFreshness(lastKnown.FetchedAt))
		_ = qEng.SaveUsage(lastKnown)
		if debug {
			slog.Debug("loadUsageSnapshot: returning lastKnown as ESTIMATED", "provider", providerName, "profile", name, "lastKnownFetchedAt", lastKnown.FetchedAt, "lastKnownAccount", lastKnown.Account)
		}
		return lastKnown
	}

	if debug {
		slog.Debug("loadUsageSnapshot: returning final snap", "provider", providerName, "profile", name, "status", snap.Status)
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
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("GetQuotaView: starting", "provider", providerName, "profile", name, "email", email)
	}

	snap := GetUsageSnapshot(providerName, name)
	if debug {
		slog.Debug("GetQuotaView: after GetUsageSnapshot", "provider", providerName, "profile", name, "status", snap.Status, "account", snap.Account, "windows", len(snap.Windows))
	}

	// A cached quota can outlive an account switch. Do not show or score data
	// whose recorded identity differs from the profile's authenticated email.
	if snapshotAccount := strings.TrimSpace(snap.Account); snapshotAccount != "" && strings.TrimSpace(email) != "" && !strings.EqualFold(snapshotAccount, strings.TrimSpace(email)) {
		if debug {
			slog.Debug("GetQuotaView: account mismatch detected", "provider", providerName, "profile", name, "snapshotAccount", snapshotAccount, "profileEmail", email)
		}
		snap.Status = model.UsageUnknown
		snap.Source = model.SourceNone
		snap.Windows = nil
		snap.Error = fmt.Sprintf("quota belongs to %s, profile is authenticated as %s", snapshotAccount, strings.TrimSpace(email))
	}

	// Stale local files must not score as fresh CACHED (e.g. August quota.json).
	qEng := quota.NewEngine(quota.DefaultTTL)
	if len(snap.Windows) > 0 && snap.Status != model.UsageUnknown && !qEng.Trustworthy(snap) {
		if debug {
			slog.Debug("GetQuotaView: marking stale as ESTIMATED", "provider", providerName, "profile", name, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
		}
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

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// resolveAGYAuthenticatedEmail reads the actual Google account email for an
// AGY profile from google_accounts.json or jetski_state.pbtxt.
// Returns "" when no authoritative email can be determined.
func resolveAGYAuthenticatedEmail(profileName string) string {
	root, err := config.ProfileRoot("agy", profileName)
	if err != nil {
		return ""
	}
	home := filepath.Join(root, "home")

	// 1. google_accounts.json in profile home.
	accountsFile := filepath.Join(home, ".gemini", "google_accounts.json")
	if email := readActiveEmailFromAccountsFile(accountsFile); email != "" {
		return email
	}

	// 2. jetski_state.pbtxt in profile home.
	jetskiFile := filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
	if email := readEmailFromJetski(jetskiFile); email != "" {
		return email
	}

	return ""
}

func readActiveEmailFromAccountsFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var acc struct {
		Active string `json:"active"`
	}
	if json.Unmarshal(data, &acc) == nil {
		return acc.Active
	}
	return ""
}

func readEmailFromJetski(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	if matches := emailRegex.FindAllString(string(data), -1); len(matches) > 0 {
		return matches[0]
	}
	return ""
}
