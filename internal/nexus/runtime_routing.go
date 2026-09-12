package nexus

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AffinityMode string

const (
	AffinityAuto   AffinityMode = "AUTO"
	AffinityPrefer AffinityMode = "PREFER"
	AffinityPin    AffinityMode = "PIN"
)

type AffinityPreference struct {
	Mode  AffinityMode `json:"mode"`
	Value string       `json:"value,omitempty"`
}

type RuntimeAffinityPolicy struct {
	Engine   AffinityPreference `json:"engine"`
	Provider AffinityPreference `json:"provider"`
	Profile  AffinityPreference `json:"profile"`
	Model    AffinityPreference `json:"model"`
}

type ModelCandidate struct {
	Provider       string   `json:"provider"`
	Profile        string   `json:"profile"`
	Model          string   `json:"model"`
	Capabilities   []string `json:"capabilities,omitempty"`
	CostRank       int      `json:"cost_rank"`
	ReasoningRank  int      `json:"reasoning_rank"`
	Healthy        bool     `json:"healthy"`
	Authenticated  bool     `json:"authenticated"`
	QuotaAvailable bool     `json:"quota_available"`
}

type RuntimeRoutingDecision struct {
	TaskID             string                `json:"task_id,omitempty"`
	PlanRevision       int                   `json:"work_plan_revision,omitempty"`
	AgentID            string                `json:"agent_id,omitempty"`
	Requirements       TaskRequirements      `json:"task_requirements"`
	Desired            RuntimeAffinityPolicy `json:"desired"`
	AffinityPolicy     RuntimeAffinityPolicy `json:"affinity_policy"`
	Actual             ModelCandidate        `json:"actual"`
	SelectedEngine     string                `json:"selected_engine,omitempty"`
	SelectedProvider   string                `json:"selected_provider,omitempty"`
	SelectedProfile    string                `json:"selected_profile,omitempty"`
	SelectedModel      string                `json:"selected_model,omitempty"`
	SelectedReasoning  string                `json:"selected_reasoning,omitempty"`
	Alternatives       []ModelCandidate      `json:"alternatives,omitempty"`
	Fallback           bool                  `json:"fallback"`
	Reason             string                `json:"reason"`
	Rejected           []string              `json:"rejected,omitempty"`
	RejectedCandidates []string              `json:"rejected_candidates,omitempty"`
	TaskClass          string                `json:"task_class,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
}

type ExecutionRoutingDecision = RuntimeRoutingDecision

var ErrBlockedResource = errors.New("blocked resource")

func ResolveRuntimeRouting(req TaskRequirements, policy RuntimeAffinityPolicy, accounts []ProviderAccount, models []ModelCandidate) (RuntimeRoutingDecision, error) {
	decision := RuntimeRoutingDecision{Desired: policy, AffinityPolicy: policy, Requirements: req, TaskClass: req.TaskKind, CreatedAt: time.Now().UTC()}
	accountCandidates := eligibleAccounts(accounts)
	if policy.Provider.Mode == AffinityPin && !containsAccount(accountCandidates, "provider", policy.Provider.Value) {
		return decision, fmt.Errorf("%w: pinned provider %q unavailable", ErrBlockedResource, policy.Provider.Value)
	}
	if policy.Profile.Mode == AffinityPin && !containsAccount(accountCandidates, "profile", policy.Profile.Value) {
		return decision, fmt.Errorf("%w: pinned profile %q unavailable", ErrBlockedResource, policy.Profile.Value)
	}
	if policy.Model.Mode == AffinityPin && !containsModel(models, policy.Model.Value) {
		return decision, fmt.Errorf("%w: pinned model %q unavailable", ErrBlockedResource, policy.Model.Value)
	}

	filtered := filterAccounts(accountCandidates, policy)
	if len(filtered) == 0 {
		if policy.Provider.Mode == AffinityPin || policy.Profile.Mode == AffinityPin {
			return decision, fmt.Errorf("%w: pinned runtime has no eligible account", ErrBlockedResource)
		}
		filtered = accountCandidates
		decision.Fallback = policy.Provider.Mode == AffinityPrefer || policy.Profile.Mode == AffinityPrefer
		if decision.Fallback {
			decision.Reason = "preferred provider/profile unavailable; scheduler fallback permitted"
		}
	}
	if len(filtered) == 0 {
		return decision, fmt.Errorf("%w: no eligible provider profile", ErrBlockedResource)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Available != filtered[j].Available {
			return filtered[i].Available
		}
		if filtered[i].QuotaRemaining != filtered[j].QuotaRemaining {
			return filtered[i].QuotaRemaining > filtered[j].QuotaRemaining
		}
		return filtered[i].Provider+":"+filtered[i].Profile < filtered[j].Provider+":"+filtered[j].Profile
	})
	chosenAccount := filtered[0]

	modelPool := filterModels(models, chosenAccount, policy)
	if len(modelPool) == 0 {
		if policy.Model.Mode == AffinityPin {
			return decision, fmt.Errorf("%w: pinned model has no eligible candidate", ErrBlockedResource)
		}
		if policy.Model.Mode == AffinityPrefer {
			decision.Fallback = true
			policy.Model = AffinityPreference{Mode: AffinityAuto}
			modelPool = filterModels(models, chosenAccount, policy)
		}
		if len(modelPool) == 0 {
			return decision, fmt.Errorf("%w: no model candidate for %s:%s", ErrBlockedResource, chosenAccount.Provider, chosenAccount.Profile)
		}
	}
	chosenModel := chooseTaskModel(modelPool, req)
	if policy.Model.Mode == AffinityPrefer && chosenModel.Model != policy.Model.Value {
		decision.Fallback = true
		decision.Reason = firstNonEmpty(decision.Reason, "preferred model unavailable; cheaper capable model selected")
	}
	decision.Actual = chosenModel
	decision.SelectedProvider = chosenModel.Provider
	decision.SelectedProfile = chosenModel.Profile
	decision.SelectedModel = chosenModel.Model
	decision.SelectedReasoning = fmt.Sprintf("cost_rank=%d; reasoning_rank=%d", chosenModel.CostRank, chosenModel.ReasoningRank)
	for _, candidate := range modelPool {
		if candidate.Provider == chosenModel.Provider && candidate.Profile == chosenModel.Profile && candidate.Model == chosenModel.Model {
			continue
		}
		decision.Alternatives = append(decision.Alternatives, candidate)
	}
	if decision.Reason == "" {
		decision.Reason = fmt.Sprintf("selected %s for task class %s using %s affinity", chosenModel.Model, req.TaskKind, affinitySummary(policy))
	}
	return decision, nil
}

func eligibleAccounts(accounts []ProviderAccount) []ProviderAccount {
	result := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if account.Authenticated && account.Available && !account.RateLimited {
			result = append(result, account)
		}
	}
	return result
}

func filterAccounts(accounts []ProviderAccount, policy RuntimeAffinityPolicy) []ProviderAccount {
	result := append([]ProviderAccount(nil), accounts...)
	if policy.Provider.Value != "" && policy.Provider.Mode != AffinityAuto {
		result = filterAccountField(result, "provider", policy.Provider.Value)
	}
	if policy.Profile.Value != "" && policy.Profile.Mode != AffinityAuto {
		result = filterAccountField(result, "profile", policy.Profile.Value)
	}
	return result
}

func filterAccountField(accounts []ProviderAccount, field, value string) []ProviderAccount {
	result := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if (field == "provider" && account.Provider == value) || (field == "profile" && account.Profile == value) {
			result = append(result, account)
		}
	}
	return result
}

func filterModels(models []ModelCandidate, account ProviderAccount, policy RuntimeAffinityPolicy) []ModelCandidate {
	result := make([]ModelCandidate, 0, len(models))
	for _, model := range models {
		if model.Provider != account.Provider || model.Profile != account.Profile || !model.Healthy || !model.Authenticated || !model.QuotaAvailable {
			continue
		}
		if policy.Model.Value != "" && policy.Model.Mode != AffinityAuto && model.Model != policy.Model.Value {
			continue
		}
		result = append(result, model)
	}
	return result
}

func chooseTaskModel(models []ModelCandidate, req TaskRequirements) ModelCandidate {
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].CostRank != models[j].CostRank {
			return models[i].CostRank < models[j].CostRank
		}
		if models[i].ReasoningRank != models[j].ReasoningRank {
			return models[i].ReasoningRank < models[j].ReasoningRank
		}
		return models[i].Model < models[j].Model
	})
	minimumReasoning := 1
	if req.TaskKind == "architecture" || req.TaskKind == "security" || req.EstimatedComplexity == "high" {
		minimumReasoning = 3
	}
	for _, model := range models {
		if model.ReasoningRank >= minimumReasoning {
			return model
		}
	}
	return models[len(models)-1]
}

func containsAccount(accounts []ProviderAccount, field, value string) bool {
	for _, account := range accounts {
		if (field == "provider" && account.Provider == value) || (field == "profile" && account.Profile == value) {
			return true
		}
	}
	return false
}

func containsModel(models []ModelCandidate, value string) bool {
	for _, model := range models {
		if model.Model == value && model.Healthy && model.Authenticated && model.QuotaAvailable {
			return true
		}
	}
	return false
}

func affinitySummary(policy RuntimeAffinityPolicy) string {
	parts := []string{string(policy.Provider.Mode), string(policy.Profile.Mode), string(policy.Model.Mode)}
	return strings.Join(parts, "/")
}
