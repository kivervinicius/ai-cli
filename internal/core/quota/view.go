package quota

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// QuotaView is the omnibus view model for quota display and scoring.
// Consumed by TUI, Web API, CLI, and Scheduler.
type QuotaView struct {
	Provider     string       `json:"provider"`
	Profile      string       `json:"profile"`
	Account      string       `json:"account"`
	Plan         string       `json:"plan"`
	Status       string       `json:"status"`
	Source       string       `json:"source"`
	ModelGroups  []ModelGroup `json:"model_groups"`
	FetchedAt    time.Time    `json:"fetched_at"`
	Available    bool         `json:"available"`
	AvailReasons AvailReasons `json:"avail_reasons"`
}

// AvailReasons holds structured reasons for availability status.
type AvailReasons struct {
	ExhaustedWindows []string `json:"exhausted_windows,omitempty"`
	RateLimited      bool     `json:"rate_limited,omitempty"`
	UnknownQuota     bool     `json:"unknown_quota,omitempty"`
	AuthRequired     bool     `json:"auth_required,omitempty"`
	AllOK            bool     `json:"all_ok,omitempty"`
}

// ModelGroup clusters usage windows by model family.
// A provider like AGY exposes two groups (Gemini, Claude/GPT).
// Simpler providers have a single unnamed group.
type ModelGroup struct {
	Name    string   `json:"name"`
	Windows []Window `json:"windows"`
}

// Window is a display-ready usage window within a model group.
type Window struct {
	Kind      string  `json:"kind"`
	Label     string  `json:"label"`
	Remaining float64 `json:"remaining"`
	ResetDesc string  `json:"reset_desc"`
	Status    string  `json:"status"`
	Bar       string  `json:"bar"`
}

// Bottleneck returns the minimum remaining percentage across ALL windows and groups.
// The second return value identifies the bottleneck window kind.
// Used by scheduler for scoring and by TUI for the inline summary bar.
func (qv *QuotaView) Bottleneck() (float64, string) {
	minPct := 100.0
	var bottleneckKind string
	found := false
	for _, g := range qv.ModelGroups {
		for _, w := range g.Windows {
			if w.Kind == "unknown" {
				continue
			}
			found = true
			if w.Remaining < minPct {
				minPct = w.Remaining
				bottleneckKind = w.Kind
			}
		}
	}
	if !found {
		return 0, ""
	}
	return minPct, bottleneckKind
}

// AllWindows returns a flat list of all windows across all groups.
// Used by scheduler for multi-window bottleneck scoring.
func (qv *QuotaView) AllWindows() []Window {
	var all []Window
	for _, g := range qv.ModelGroups {
		all = append(all, g.Windows...)
	}
	return all
}

// HasMultipleGroups returns true when the provider exposes separate model families.
func (qv *QuotaView) HasMultipleGroups() bool {
	return len(qv.ModelGroups) > 1
}

// IsAvailable returns true if the agent can accept work RIGHT NOW.
// An agent is unavailable when ANY quota window is at 0% (exhausted)
// or when rate-limited.
// UNKNOWN quota is NOT a hard block — we just don't know, so we penalize
// the score instead of rejecting.
func (qv *QuotaView) IsAvailable() bool {
	return qv.Available
}

// AvailabilityLabel returns a human-readable availability status.
func (qv *QuotaView) AvailabilityLabel() string {
	if qv.Available {
		return "DISPONIVEL"
	}
	if qv.AvailReasons.RateLimited {
		return "RATE LIMITED"
	}
	if len(qv.AvailReasons.ExhaustedWindows) > 0 {
		return "QUOTA ESGOTADA"
	}
	if qv.AvailReasons.AuthRequired {
		return "SEM AUTENTICACAO"
	}
	return "INDISPONIVEL"
}

// ComputeAvailability analyzes all windows and sets Available + AvailReasons.
func (qv *QuotaView) ComputeAvailability() {
	var exhausted []string
	qv.Available = true
	qv.AvailReasons = AvailReasons{}

	// Check rate-limited status — hard filter.
	if qv.Status == string(model.UsageRateLimited) {
		qv.Available = false
		qv.AvailReasons.RateLimited = true
		return
	}

	// Check auth required — hard filter.
	if qv.Status == "auth_required" {
		qv.Available = false
		qv.AvailReasons.AuthRequired = true
		return
	}

	// Check UNKNOWN quota — NOT a hard block, just flag it.
	if qv.Status == string(model.UsageUnknown) {
		qv.AvailReasons.UnknownQuota = true
		// Don't set Available=false here — UNKNOWN means we don't know,
		// not that it's exhausted. The scheduler penalizes this in scoring.
	}

	// Scan ALL windows across ALL groups for exhausted quotas (0% = hard filter).
	// Skip "unknown" kind — it's a placeholder for missing data, not a real quota.
	for _, g := range qv.ModelGroups {
		for _, w := range g.Windows {
			if w.Kind == "unknown" {
				continue
			}
			if w.Remaining <= 0.0 {
				exhausted = append(exhausted, w.Kind)
			}
		}
	}

	if len(exhausted) > 0 {
		qv.Available = false
		qv.AvailReasons.ExhaustedWindows = exhausted
		return
	}

	// All checks passed.
	qv.AvailReasons.AllOK = true
}

// RenderBar renders a progress bar for a given remaining percentage.
func RenderBar(remaining float64, status string, width int) string {
	var pctPtr *float64
	if status == string(model.UsageLive) || status == string(model.UsageCached) || status == string(model.UsageEstimated) {
		pctPtr = &remaining
	}
	return RenderProgressBar(model.UsageStatus(status), pctPtr, width)
}

// RenderBarWithPercent renders a progress bar with a percentage label.
func RenderBarWithPercent(remaining float64, status string, width int) string {
	bar := RenderBar(remaining, status, width)
	if status == string(model.UsageLive) || status == string(model.UsageCached) || status == string(model.UsageEstimated) {
		return fmt.Sprintf("%s %2.0f%%", bar, remaining)
	}
	return bar
}

// WindowLabel returns the display label for a window kind.
func WindowLabel(kind string) string {
	switch kind {
	case "5h", "daily":
		return "Limite 5 Horas"
	case "weekly":
		return "Limite Semanal"
	case "claude_5h", "claude_five_hour":
		return "Limite 5 Horas"
	case "claude_weekly":
		return "Limite Semanal"
	default:
		return kind
	}
}

// GroupDisplayName returns the display name for a model group.
func GroupDisplayName(groupKey string) string {
	switch groupKey {
	case "gemini":
		return "Gemini Models"
	case "claude_gpt":
		return "Claude & GPT Models"
	case "":
		return ""
	default:
		return groupKey
	}
}

// sortGroupOrder defines a stable display order for model groups.
var sortGroupOrder = map[string]int{
	"":           0,
	"gemini":     1,
	"claude_gpt": 2,
}

// SortModelGroups orders model groups for consistent display.
func SortModelGroups(groups []ModelGroup) {
	sort.Slice(groups, func(i, j int) bool {
		oi, okI := sortGroupOrder[groups[i].Name]
		oj, okJ := sortGroupOrder[groups[j].Name]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return groups[i].Name < groups[j].Name
	})
}

// GroupKeys returns a sorted slice of unique group keys from windows.
func GroupKeys(windows []model.UsageWindow) []string {
	seen := make(map[string]bool)
	for _, w := range windows {
		seen[w.Group] = true
	}
	var keys []string
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		oi, okI := sortGroupOrder[keys[i]]
		oj, okJ := sortGroupOrder[keys[j]]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}

// NormalizeGroupLabel removes trailing whitespace and normalizes group names for display.
func NormalizeGroupLabel(s string) string {
	return strings.TrimSpace(s)
}
