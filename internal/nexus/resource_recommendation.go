package nexus

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// TaskRequirements specifies what an intelligence or execution task demands (§Gate 5, Phase B).
type TaskRequirements struct {
	TaskKind             string   `json:"task_kind"` // "coding" | "planning" | "review" | "verify" | "refactor" | "security"
	Role                 string   `json:"role"`      // "implementer" | "reviewer" | "architect" | "tester"
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	EstimatedTokens      int      `json:"estimated_tokens,omitempty"`
	CurrentProvider      string   `json:"current_provider,omitempty"`
	CurrentProfile       string   `json:"current_profile,omitempty"`
	PreferProvider       string   `json:"prefer_provider,omitempty"`
	ProjectPolicy        string   `json:"project_policy,omitempty"`
	AgentPreference      string   `json:"agent_preference,omitempty"`
	ProviderLock         string   `json:"provider_lock,omitempty"`
	ProfileLock          string   `json:"profile_lock,omitempty"`
}

// ResourceCandidate represents an evaluated provider account scored for a specific task.
type ResourceCandidate struct {
	Account         ProviderAccount    `json:"account"`
	Rank            int                `json:"rank"`
	TotalScore      float64            `json:"total_score"`
	Confidence      string             `json:"confidence"` // "LIVE" | "CACHED" | "ESTIMATED" | "UNKNOWN"
	ScoreBreakdown  map[string]float64 `json:"score_breakdown"`
	Pros            []string           `json:"pros"`
	Cons            []string           `json:"cons"`
	Eligible        bool               `json:"eligible"`
	RejectionReason string             `json:"rejection_reason,omitempty"`
}

// RecommendationResult holds the ranked candidates and top selection with explainable reasoning.
type RecommendationResult struct {
	Requirements TaskRequirements    `json:"requirements"`
	Policy       SchedulerPolicy     `json:"policy"`
	Recommended  *ResourceCandidate  `json:"recommended,omitempty"`
	Candidates   []ResourceCandidate `json:"candidates"`
	Explanation  string              `json:"explanation"`
}

// RecommendResources evaluates available accounts against task requirements under the chosen policy.
func RecommendResources(accounts []ProviderAccount, req TaskRequirements, policy SchedulerPolicy) RecommendationResult {
	if policy == "" {
		if req.ProjectPolicy != "" {
			policy = SchedulerPolicy(strings.ToUpper(req.ProjectPolicy))
		} else {
			policy = PolicyBalanced
		}
	}

	result := RecommendationResult{
		Requirements: req,
		Policy:       policy,
		Candidates:   make([]ResourceCandidate, 0, len(accounts)),
	}

	for _, acc := range accounts {
		c := evaluateCandidate(acc, req, policy)
		result.Candidates = append(result.Candidates, c)
	}

	// Sort candidates: eligible first, then higher score, then lower rate limit risk
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Eligible != result.Candidates[j].Eligible {
			return result.Candidates[i].Eligible
		}
		if math.Abs(result.Candidates[i].TotalScore-result.Candidates[j].TotalScore) > 0.001 {
			return result.Candidates[i].TotalScore > result.Candidates[j].TotalScore
		}
		return result.Candidates[i].Account.DisplayName < result.Candidates[j].Account.DisplayName
	})

	// Assign ranks and identify recommended
	for i := range result.Candidates {
		result.Candidates[i].Rank = i + 1
		if result.Candidates[i].Eligible && result.Recommended == nil {
			cand := result.Candidates[i]
			result.Recommended = &cand
		}
	}

	if result.Recommended != nil {
		result.Explanation = fmt.Sprintf("Recomendado %s (%s) com score %.1f baseado na política %s.",
			result.Recommended.Account.DisplayName, result.Recommended.Account.Provider, result.Recommended.TotalScore, policy)
	} else {
		result.Explanation = "Nenhum provedor elegível atende aos requisitos da tarefa (todos indisponíveis ou sem autenticação)."
	}

	return result
}

func evaluateCandidate(acc ProviderAccount, req TaskRequirements, policy SchedulerPolicy) ResourceCandidate {
	c := ResourceCandidate{
		Account:        acc,
		ScoreBreakdown: make(map[string]float64),
		Pros:           make([]string, 0),
		Cons:           make([]string, 0),
		Eligible:       true,
		Confidence:     "UNKNOWN",
	}

	// Hard Gate 0: an explicit provider/profile lock is authoritative.
	if lock := strings.TrimSpace(req.ProviderLock); lock != "" && !strings.EqualFold(acc.Provider, lock) {
		c.Eligible = false
		c.RejectionReason = fmt.Sprintf("Provider bloqueado para %s", lock)
		c.Cons = append(c.Cons, c.RejectionReason)
		return c
	}
	if lock := strings.TrimSpace(req.ProfileLock); lock != "" && acc.Profile != lock {
		c.Eligible = false
		c.RejectionReason = fmt.Sprintf("Profile bloqueado para %s", lock)
		c.Cons = append(c.Cons, c.RejectionReason)
		return c
	}

	// Hard Gate 1: Must be authenticated
	if !acc.Authenticated {
		c.Eligible = false
		c.RejectionReason = "Conta não autenticada no sistema"
		c.Cons = append(c.Cons, "Não autenticada")
		return c
	}

	// Hard Gate 2: Must be available / not in active cooldown
	if !acc.Available {
		c.Eligible = false
		c.RejectionReason = "Provedor marcado como indisponível"
		c.Cons = append(c.Cons, "Indisponível")
		return c
	}

	if acc.RateLimited {
		c.Eligible = false
		c.RejectionReason = "Conta em rate limit / cooldown ativo"
		c.Cons = append(c.Cons, "Rate limit ativo")
		return c
	}

	if acc.CooldownUntil != nil && time.Now().Before(*acc.CooldownUntil) {
		c.Eligible = false
		c.RejectionReason = fmt.Sprintf("Em cooldown até %s", acc.CooldownUntil.Format(time.RFC3339))
		c.Cons = append(c.Cons, "Em cooldown")
		return c
	}

	// Hard Gate 3: every required capability must be explicitly SUPPORTED.
	// PARTIAL/UNKNOWN/NOT_TESTED are not enough for a hard requirement.
	for _, required := range req.RequiredCapabilities {
		key := normalizeCapabilityName(required)
		status := strings.ToUpper(strings.TrimSpace(acc.Capabilities[key]))
		if status != "SUPPORTED" {
			c.Eligible = false
			c.RejectionReason = fmt.Sprintf("Capacidade obrigatória %s não suportada (%s)", key, capabilityStatusLabel(status))
			c.Cons = append(c.Cons, c.RejectionReason)
			return c
		}
	}

	// Determine quota confidence from the actual usage status. A non-nil
	// QuotaView is not evidence that the data is LIVE.
	c.Confidence = quotaConfidence(acc)

	var score float64 = 50.0 // Baseline

	// 1. Quota & Capacity Scoring (0 to 30 pts)
	quotaScore := 0.0
	switch c.Confidence {
	case "LIVE":
		quotaScore = acc.QuotaRemaining * 30.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota em tempo real (%.0f%% restante)", acc.QuotaRemaining*100))
	case "CACHED":
		quotaScore = acc.QuotaRemaining * 20.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota em cache (%.0f%% restante)", acc.QuotaRemaining*100))
	case "ESTIMATED":
		quotaScore = acc.QuotaRemaining * 15.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota estimada (%.0f%% restante)", acc.QuotaRemaining*100))
	case "UNKNOWN":
		// Unknown quota is NOT treated as best (§Phase B: Unknown quota must not be treated as magically best)
		quotaScore = 10.0
		c.Cons = append(c.Cons, "Quota não aferida em tempo real")
	}
	c.ScoreBreakdown["quota"] = quotaScore
	score += quotaScore

	// 2. Health & Reliability (0 to 20 pts)
	healthScore := 10.0
	switch acc.Health {
	case "healthy":
		healthScore = 20.0
		c.Pros = append(c.Pros, "Saúde operacional excelente")
	case "degraded":
		healthScore = 5.0
		c.Cons = append(c.Cons, "Saúde degradada")
	case "unhealthy":
		healthScore = 0.0
		c.Cons = append(c.Cons, "Falhas recentes registradas")
	}
	c.ScoreBreakdown["health"] = healthScore
	score += healthScore

	// 3. Affinity & Continuity (0 to 20 pts)
	affinityScore := 0.0
	if req.CurrentProvider != "" && acc.Provider == req.CurrentProvider {
		affinityScore += 15.0
		c.Pros = append(c.Pros, "Afinidade com sessão/runtime atual (sem custo de troca de contexto)")
		if req.CurrentProfile != "" && acc.Profile == req.CurrentProfile {
			affinityScore += 5.0
			c.Pros = append(c.Pros, "Perfil idêntico ao runtime ativo")
		}
	} else if req.CurrentProvider != "" {
		c.Cons = append(c.Cons, "Exige troca de provedor (custo de novo contexto)")
	}
	c.ScoreBreakdown["affinity"] = affinityScore
	score += affinityScore

	// 4. Role & Capability Fit (0 to 20 pts). Never infer competence from
	// provider brand/name; drivers publish the capabilities used here.
	roleScore := 10.0
	supported := func(name string) bool {
		return strings.EqualFold(strings.TrimSpace(acc.Capabilities[normalizeCapabilityName(name)]), "SUPPORTED")
	}
	switch req.Role {
	case "reviewer", "tester":
		if supported("read_only_review") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Capability explícita para revisão independente/read-only")
		}
	case "implementer":
		if supported("autonomous_coding") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Capability explícita para implementação autônoma")
		}
	case "architect", "planner":
		if supported("headless") && supported("submit_prompt") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Capabilities explícitas para planejamento headless")
		}
	}
	c.ScoreBreakdown["role_fit"] = roleScore
	score += roleScore

	// 5. Policy Multipliers
	switch policy {
	case PolicyPreserveQuota:
		if acc.QuotaRemaining > 0.7 {
			score += 25.0
			c.Pros = append(c.Pros, "Preserva quotas escassas utilizando conta com alta disponibilidade")
		} else if acc.QuotaRemaining < 0.3 && c.Confidence == "LIVE" {
			score -= 20.0
			c.Cons = append(c.Cons, "Penalizado pela política Preserve Quota (quota < 30%)")
		}
	case PolicyPreferProvider:
		prefer := req.PreferProvider
		if prefer == "" {
			prefer = req.AgentPreference
		}
		if prefer != "" && (acc.Provider == prefer || acc.ID == prefer) {
			score += 35.0
			c.Pros = append(c.Pros, fmt.Sprintf("Provedor preferencial explícito (%s)", prefer))
		}
	case PolicyManual:
		if req.PreferProvider != "" && acc.Provider == req.PreferProvider {
			score += 50.0
		} else {
			score -= 50.0
		}
	case PolicyBalanced:
		// Default balanced weights
	}

	c.TotalScore = math.Round(score*10) / 10
	return c
}

func quotaConfidence(acc ProviderAccount) string {
	if acc.QuotaView != nil {
		switch strings.ToUpper(strings.TrimSpace(acc.QuotaView.Status)) {
		case "LIVE":
			return "LIVE"
		case "CACHED":
			return "CACHED"
		case "ESTIMATED":
			return "ESTIMATED"
		default:
			return "UNKNOWN"
		}
	}
	if acc.QuotaRemaining > 0 {
		return "CACHED"
	}
	return "UNKNOWN"
}

func normalizeCapabilityName(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func capabilityStatusLabel(status string) string {
	if status == "" {
		return "UNKNOWN"
	}
	return status
}
