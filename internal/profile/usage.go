package profile

import (
	"context"
	"fmt"
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
func GetUsageSnapshot(providerName, name string) model.UsageSnapshot {
	qEng := quota.NewEngine(5 * time.Minute)
	snap, found := qEng.GetCachedUsage(providerName, name)
	if !found || snap.Status == model.UsageUnknown {
		ctx := context.Background()
		p := model.Profile{Provider: providerName, Name: name}
		switch providerName {
		case "codex":
			snap = codex.New().GetUsage(ctx, p)
		case "agy":
			snap = agy.New().GetUsage(ctx, p)
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
		if providerName == "codex" {
			q.ModelName = "gpt-5.6-sol"
		} else if providerName == "agy" {
			q.ModelName = "Gemini 2.5 Flash / Pro"
		} else if providerName == "claude" {
			q.ModelName = "Claude 3.7 Sonnet"
		} else if providerName == "opencode" {
			q.ModelName = "OpenCode Provider"
		} else if providerName == "gemini" {
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
		if w.Kind == "5h" || w.Kind == "daily" {
			q.FiveHour = lw
		} else if w.Kind == "weekly" {
			q.Weekly = lw
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

// RenderBar generates a clean progress bar.
func RenderBar(percent float64, width int) string {
	p := percent
	return quota.RenderProgressBar(model.UsageLive, &p, width)
}

func RenderShortBar(percent float64) string {
	p := percent
	return quota.RenderShortStatus(model.UsageLive, &p, 10)
}
