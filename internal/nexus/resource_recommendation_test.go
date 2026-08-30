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
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
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
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
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
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
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
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
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

func TestResourceRecommendation_UnknownQuotaDoesNotOutrankKnownLiveQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "unknown",
			Provider:       "codex",
			Profile:        "unknown",
			DisplayName:    "Unknown quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "UNKNOWN"},
			QuotaRemaining: 0,
			Health:         "healthy",
		},
		{
			ID:             "known",
			Provider:       "claude",
			Profile:        "known",
			DisplayName:    "Known quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.50,
			Health:         "healthy",
		},
	}

	res := RecommendResources(accounts, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.ID != "known" {
		t.Fatalf("known LIVE quota must outrank UNKNOWN quota, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.ID == "unknown" && candidate.Confidence != "UNKNOWN" {
			t.Fatalf("unknown quota confidence = %q, want UNKNOWN", candidate.Confidence)
		}
	}
}

func TestResourceRecommendation_RequiredCapabilitiesAreHardGate(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "no-resume",
			Provider:       "agy",
			Profile:        "default",
			DisplayName:    "AGY",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.95,
			Health:         "healthy",
			Capabilities: map[string]string{
				"resume": "UNSUPPORTED",
			},
		},
		{
			ID:             "has-resume",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "Codex",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.40,
			Health:         "healthy",
			Capabilities: map[string]string{
				"resume": "SUPPORTED",
			},
		},
	}

	res := RecommendResources(accounts, TaskRequirements{RequiredCapabilities: []string{"resume"}}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.ID != "has-resume" {
		t.Fatalf("required capability must reject unsupported candidate, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.ID == "no-resume" && candidate.Eligible {
			t.Fatal("candidate without required capability must be ineligible")
		}
	}
}

func TestResourceRecommendation_ProviderProfileLockIsHardGate(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex", Authenticated: true, Available: true, Health: "healthy", Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"}},
		{ID: "claude-review", Provider: "claude", Profile: "review", DisplayName: "Claude", Authenticated: true, Available: true, Health: "healthy", Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"}},
	}
	res := RecommendResources(accounts, TaskRequirements{Role: "implementer", ProviderLock: "claude", ProfileLock: "review", RequiredCapabilities: []string{"headless", "submit_prompt"}}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.Provider != "claude" || res.Recommended.Account.Profile != "review" {
		t.Fatalf("expected locked claude/review resource, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.Provider == "codex" && candidate.Eligible {
			t.Fatalf("provider lock must hard-reject codex: %#v", candidate)
		}
	}
}

func TestResourceRecommendation_RoleFitUsesCapabilitiesNotProviderName(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID: "future-review", Provider: "future-ai", Profile: "default", DisplayName: "Future AI",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.5,
			Capabilities: map[string]string{"read_only_review": "SUPPORTED"},
		},
		{
			ID: "claude-no-review", Provider: "claude", Profile: "default", DisplayName: "Claude without review",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.5,
			Capabilities: map[string]string{"read_only_review": "UNSUPPORTED"},
		},
	}
	res := RecommendResources(accounts, TaskRequirements{Role: "reviewer"}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.ID != "future-review" {
		t.Fatalf("role fit must follow explicit capabilities, got %#v", res.Recommended)
	}
}
