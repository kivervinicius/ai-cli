package nexus

import (
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

func TestResourceRecommendation_Balanced(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "codex-default",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "OpenAI Codex (Default)",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{},
			QuotaRemaining: 0.85,
			Health:         "healthy",
		},
		{
			ID:             "claude-default",
			Provider:       "claude",
			Profile:        "default",
			DisplayName:    "Anthropic Claude",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{},
			QuotaRemaining: 0.40,
			Health:         "healthy",
		},
		{
			ID:            "gemini-unauthed",
			Provider:      "gemini",
			Profile:       "default",
			DisplayName:   "Google Gemini",
			Authenticated: false,
			Available:     false,
			Health:        "unknown",
		},
	}

	req := TaskRequirements{
		TaskKind: "coding",
		Role:     "implementer",
	}

	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil {
		t.Fatalf("expected recommended account, got nil")
	}

	if res.Recommended.Account.Provider != "codex" {
		t.Errorf("expected codex to be recommended due to higher quota, got %s", res.Recommended.Account.Provider)
	}

	if len(res.Candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(res.Candidates))
	}

	// Unauthenticated account must be ineligible
	for _, c := range res.Candidates {
		if c.Account.Provider == "gemini" && c.Eligible {
			t.Errorf("unauthenticated gemini must be marked ineligible")
		}
	}
}

func TestResourceRecommendation_PreserveQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "codex-low",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "Codex Low Quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{},
			QuotaRemaining: 0.15, // < 30%
			Health:         "healthy",
		},
		{
			ID:             "claude-high",
			Provider:       "claude",
			Profile:        "default",
			DisplayName:    "Claude High Quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{},
			QuotaRemaining: 0.90, // > 70%
			Health:         "healthy",
		},
	}

	req := TaskRequirements{
		TaskKind: "coding",
		Role:     "implementer",
	}

	res := RecommendResources(accounts, req, PolicyPreserveQuota)
	if res.Recommended == nil || res.Recommended.Account.Provider != "claude" {
		t.Errorf("PreserveQuota policy must choose claude with 90%% quota over codex with 15%%, got %v", res.Recommended)
	}
}

func TestResourceRecommendation_CooldownRejection(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	accounts := []ProviderAccount{
		{
			ID:            "codex-cooldown",
			Provider:      "codex",
			Profile:       "default",
			DisplayName:   "Codex in Cooldown",
			Authenticated: true,
			Available:     true,
			CooldownUntil: &future,
			Health:        "degraded",
		},
	}

	res := RecommendResources(accounts, TaskRequirements{}, PolicyBalanced)
	if res.Recommended != nil {
		t.Errorf("account in cooldown must not be recommended")
	}
	if res.Candidates[0].Eligible {
		t.Errorf("account in cooldown must be marked ineligible")
	}
}
