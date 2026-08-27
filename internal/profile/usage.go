package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LimitWindow struct {
	PercentLeft float64 `json:"percent_left"`
	ResetsIn    string  `json:"resets_in"`
	ResetTime   string  `json:"reset_time"`
	ProgressBar string  `json:"progress_bar"`
}

type QuotaDetails struct {
	Provider    string      `json:"provider"`
	ProfileName string      `json:"profile_name"`
	Account     string      `json:"account"`
	Plan        string      `json:"plan"`
	ModelName   string      `json:"model_name"`
	FiveHour    LimitWindow `json:"five_hour"`
	Weekly      LimitWindow `json:"weekly"`
	ClaudeFiveH LimitWindow `json:"claude_five_hour,omitempty"`
	ClaudeWeek  LimitWindow `json:"claude_weekly,omitempty"`
}

// GetQuotaDetails returns the isolated 5h and weekly quota metrics for a profile.
func GetQuotaDetails(provider, name, plan, email string) QuotaDetails {
	q := QuotaDetails{
		Provider:    provider,
		ProfileName: name,
		Account:     email,
		Plan:        plan,
	}

	// Try reading cached/saved profile quota file
	root, err := Root(provider, name)
	quotaFile := ""
	if err == nil {
		quotaFile = filepath.Join(root, "quota.json")
		if data, err := os.ReadFile(quotaFile); err == nil {
			var saved QuotaDetails
			if json.Unmarshal(data, &saved) == nil && saved.FiveHour.ProgressBar != "" {
				saved.Account = email
				saved.Plan = plan
				return saved
			}
		}
	}

	if provider == "codex" {
		q.ModelName = "gpt-5.6-sol (reasoning low, summaries auto)"
		q.FiveHour = LimitWindow{
			PercentLeft: 100.0,
			ResetTime:   "Quota available",
			ProgressBar: RenderBar(100.0, 20),
		}
		q.Weekly = LimitWindow{
			PercentLeft: 100.0,
			ResetTime:   "Quota available",
			ProgressBar: RenderBar(100.0, 20),
		}
	} else if provider == "agy" {
		q.ModelName = "Gemini Flash, Gemini Pro"
		q.FiveHour = LimitWindow{
			PercentLeft: 100.0,
			ResetsIn:    "Quota available",
			ProgressBar: RenderBar(100.0, 50),
		}
		q.Weekly = LimitWindow{
			PercentLeft: 100.0,
			ResetsIn:    "Quota available",
			ProgressBar: RenderBar(100.0, 50),
		}
		q.ClaudeFiveH = LimitWindow{
			PercentLeft: 100.0,
			ResetsIn:    "Quota available",
			ProgressBar: RenderBar(100.0, 50),
		}
		q.ClaudeWeek = LimitWindow{
			PercentLeft: 100.0,
			ResetsIn:    "Quota available",
			ProgressBar: RenderBar(100.0, 50),
		}
	}

	// Persist per-profile quota if directory exists
	if quotaFile != "" {
		if data, err := json.MarshalIndent(q, "", "  "); err == nil {
			_ = os.WriteFile(quotaFile, data, 0600)
		}
	}

	return q
}

// RenderBar generates a clean progress bar of the given percentage and width.
func RenderBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int((percent * float64(width)) / 100.0)
	if filled > width {
		filled = width
	}
	empty := width - filled
	if empty < 0 {
		empty = 0
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func RenderShortBar(percent float64) string {
	return RenderBar(percent, 20) + fmt.Sprintf(" %.0f%%", percent)
}
