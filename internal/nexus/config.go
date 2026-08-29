package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// AgentConfig is the persistent, revisioned configuration for an Agent (Gate 3).
// It is stored as a JSON blob in agent_revisions.config.
type AgentConfig struct {
	Provider         string            `json:"provider"`
	Profile          string            `json:"profile"`
	Model            string            `json:"model,omitempty"`
	Options          map[string]any    `json:"options,omitempty"`
	Workspace        string            `json:"workspace,omitempty"`
	Isolation        string            `json:"isolation,omitempty"`        // "project" | "global" | "none"
	MaestroMode      string            `json:"maestro_mode,omitempty"`    // "OFF" | "ASSIST" | "ORCHESTRATE"
	ContinuityPolicy string            `json:"continuity_policy,omitempty"` // "auto" | "native" | "new_session"
	Environment      map[string]string `json:"environment,omitempty"`
	Allocation       *AllocationPolicy `json:"allocation,omitempty"`
}

// AllocationPolicy controls how provider resources are allocated to an agent.
type AllocationPolicy struct {
	PreferProvider   string `json:"prefer_provider,omitempty"`
	MaxConcurrent    int    `json:"max_concurrent,omitempty"`
	QuotaPreserve    bool   `json:"quota_preserve,omitempty"`
	CooldownSeconds  int    `json:"cooldown_seconds,omitempty"`
}

// OptionSpec describes a single configuration option owned by a provider driver.
// The UI renders these generically — no provider-specific React conditionals.
type OptionSpec struct {
	Key              string   `json:"key"`
	Label            string   `json:"label"`
	Description      string   `json:"description,omitempty"`
	Type             string   `json:"type"` // "string" | "number" | "boolean" | "select" | "multiselect" | "text"
	Default          any      `json:"default,omitempty"`
	Choices          []Choice `json:"choices,omitempty"`
	Validation       string   `json:"validation,omitempty"`       // regex or rule name
	Sensitive        bool     `json:"sensitive,omitempty"`
	Advanced         bool     `json:"advanced,omitempty"`
	RestartRequired  bool     `json:"restart_required,omitempty"` // continuity impact
	Group            string   `json:"group,omitempty"`            // UI section grouping
}

// Choice is a labeled value for select/multiselect options.
type Choice struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ImpactMode describes what happens when a config is applied.
type ImpactMode string

const (
	ImpactLiveSameRuntime ImpactMode = "LIVE_SAME_RUNTIME" // No restart needed
	ImpactRestartRuntime  ImpactMode = "RESTART_RUNTIME"   // New generation, same agent
	ImpactNewSession      ImpactMode = "NEW_SESSION"       // New provider session
)

// ConfigImpact describes the result of comparing current vs proposed config.
type ConfigImpact struct {
	Mode             ImpactMode     `json:"mode"`
	ChangedFields    []string       `json:"changed_fields"`
	RequiresRestart  bool           `json:"requires_restart"`
	RequiresNewSess  bool           `json:"requires_new_session"`
	CurrentConfig    *AgentConfig   `json:"current_config,omitempty"`
	ProposedConfig   *AgentConfig   `json:"proposed_config,omitempty"`
	Warnings         []string       `json:"warnings,omitempty"`
}

// ParseAgentConfig deserializes a JSON config string into AgentConfig.
func ParseAgentConfig(raw string) (AgentConfig, error) {
	var cfg AgentConfig
	if raw == "" || raw == "{}" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

// ConfigJSON serializes AgentConfig to JSON string.
func (c AgentConfig) ConfigJSON() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// AnalyzeImpact compares current and proposed agent configs and returns the
// impact mode and changed fields. This drives the Safe Apply preview.
func AnalyzeImpact(current, proposed AgentConfig) ConfigImpact {
	impact := ConfigImpact{
		CurrentConfig:  &current,
		ProposedConfig: &proposed,
	}

	if reflect.DeepEqual(current, proposed) {
		return impact // no changes
	}

	// Detect which fields changed.
	if current.Provider != proposed.Provider {
		impact.ChangedFields = append(impact.ChangedFields, "provider")
		impact.RequiresNewSess = true
	}
	if current.Profile != proposed.Profile {
		impact.ChangedFields = append(impact.ChangedFields, "profile")
		impact.RequiresNewSess = true
	}
	if current.Model != proposed.Model {
		impact.ChangedFields = append(impact.ChangedFields, "model")
		impact.RequiresRestart = true
	}
	if current.Workspace != proposed.Workspace {
		impact.ChangedFields = append(impact.ChangedFields, "workspace")
		impact.RequiresRestart = true
	}
	if current.Isolation != proposed.Isolation {
		impact.ChangedFields = append(impact.ChangedFields, "isolation")
		impact.RequiresRestart = true
	}
	if current.MaestroMode != proposed.MaestroMode {
		impact.ChangedFields = append(impact.ChangedFields, "maestro_mode")
		// Maestro mode change is live — no restart needed.
	}
	if current.ContinuityPolicy != proposed.ContinuityPolicy {
		impact.ChangedFields = append(impact.ChangedFields, "continuity_policy")
	}
	if !reflect.DeepEqual(current.Options, proposed.Options) {
		impact.ChangedFields = append(impact.ChangedFields, "options")
		impact.RequiresRestart = true
	}
	if !reflect.DeepEqual(current.Environment, proposed.Environment) {
		impact.ChangedFields = append(impact.ChangedFields, "environment")
		impact.RequiresRestart = true
	}
	if !reflect.DeepEqual(current.Allocation, proposed.Allocation) {
		impact.ChangedFields = append(impact.ChangedFields, "allocation")
	}

	// Determine impact mode.
	switch {
	case impact.RequiresNewSess:
		impact.Mode = ImpactNewSession
	case impact.RequiresRestart:
		impact.Mode = ImpactRestartRuntime
	default:
		impact.Mode = ImpactLiveSameRuntime
	}

	return impact
}

// SafeApply applies a config change atomically: creates revision, analyzes
// impact, and if the agent has a live runtime that needs restart, performs
// the restart. Returns the impact for UI preview.
func (n *Nexus) SafeApply(ctx context.Context, agentID string, proposed AgentConfig) (*ConfigImpact, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}

	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}

	// Load current config from the agent's current revision.
	var current AgentConfig
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			current, _ = ParseAgentConfig(rev.Config)
		}
	}

	// Analyze impact.
	impact := AnalyzeImpact(current, proposed)
	if len(impact.ChangedFields) == 0 {
		return &impact, nil // no-op
	}

	// Create new immutable config revision (N+1).
	rev, err := st.AddRevision(agentID, proposed.ConfigJSON())
	if err != nil {
		return nil, fmt.Errorf("create config revision: %w", err)
	}

	// Update agent's current revision.
	agent.CurrentRevisionID = rev.ID
	if err := st.UpdateAgent(agent); err != nil {
		return nil, fmt.Errorf("update agent revision: %w", err)
	}

	// If the agent has a live runtime and the config requires restart, do it.
	if impact.RequiresRestart || impact.RequiresNewSess {
		gen, gerr := st.CurrentGeneration(agentID)
		if gerr == nil && n.runtimeAlive(gen.RuntimeID) {
			// Stop the current runtime.
			if err := n.StopAgent(ctx, agentID); err != nil {
				return nil, fmt.Errorf("stop for config apply: %w", err)
			}

			// Start with new config.
			profile := proposed.Profile
			if profile == "" {
				profile = "default"
			}
			sess, err := n.StartAgent(ctx, agentID, proposed.Provider, profile)
			if err != nil {
				return nil, fmt.Errorf("restart for config apply: %w", err)
			}
			_ = sess // runtime is now running with new config
		}
	}

	return &impact, nil
}
