package nexus

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// --- Gate 3: AgentConfig + Safe Apply ---

func TestParseAgentConfig(t *testing.T) {
	raw := `{"provider":"claude","profile":"default","model":"sonnet","options":{"temperature":0.7}}`
	cfg, err := ParseAgentConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "claude" {
		t.Errorf("provider = %q, want claude", cfg.Provider)
	}
	if cfg.Model != "sonnet" {
		t.Errorf("model = %q, want sonnet", cfg.Model)
	}
	if cfg.Options["temperature"] != 0.7 {
		t.Errorf("options.temperature = %v, want 0.7", cfg.Options["temperature"])
	}
}

func TestParseAgentConfigEmpty(t *testing.T) {
	cfg, err := ParseAgentConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "" {
		t.Errorf("empty config should have empty provider, got %q", cfg.Provider)
	}
}

func TestAnalyzeImpactNoChange(t *testing.T) {
	cfg := AgentConfig{Provider: "claude", Profile: "default"}
	impact := AnalyzeImpact(cfg, cfg)
	if len(impact.ChangedFields) != 0 {
		t.Errorf("no change should produce empty changed fields, got %v", impact.ChangedFields)
	}
	if impact.Mode != "" {
		t.Errorf("no change should have empty mode, got %q", impact.Mode)
	}
}

func TestAnalyzeImpactProviderChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", Profile: "default"}
	proposed := AgentConfig{Provider: "openai", Profile: "default"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactNewSession {
		t.Errorf("provider change should be NEW_SESSION, got %q", impact.Mode)
	}
	if !impact.RequiresNewSess {
		t.Error("provider change should require new session")
	}
	found := false
	for _, f := range impact.ChangedFields {
		if f == "provider" {
			found = true
		}
	}
	if !found {
		t.Error("changed fields should include 'provider'")
	}
}

func TestAnalyzeImpactModelChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", Model: "sonnet"}
	proposed := AgentConfig{Provider: "claude", Model: "opus"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactRestartRuntime {
		t.Errorf("model change should be RESTART_RUNTIME, got %q", impact.Mode)
	}
	if !impact.RequiresRestart {
		t.Error("model change should require restart")
	}
}

func TestAnalyzeImpactMaestroModeChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", MaestroMode: "OFF"}
	proposed := AgentConfig{Provider: "claude", MaestroMode: "ASSIST"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactLiveSameRuntime {
		t.Errorf("maestro mode change should be LIVE_SAME_RUNTIME, got %q", impact.Mode)
	}
	if impact.RequiresRestart || impact.RequiresNewSess {
		t.Error("maestro mode change should not require restart or new session")
	}
}

func TestSafeApplyCreatesRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	proposed := AgentConfig{Provider: "fake", Profile: "default", Model: "test"}
	impact, err := n.SafeApply(context.Background(), agent.ID, proposed)
	if err != nil {
		t.Fatalf("SafeApply: %v", err)
	}
	if len(impact.ChangedFields) == 0 {
		t.Error("expected changed fields after first config apply")
	}

	// Verify revision was created.
	agent, _ = st.GetAgent(agent.ID, "")
	if agent.CurrentRevisionID == "" {
		t.Fatal("agent should have a current revision after SafeApply")
	}
	rev, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseAgentConfig(rev.Config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "fake" || cfg.Model != "test" {
		t.Errorf("revision config = %+v, want provider=fake model=test", cfg)
	}
}

func TestSafeApplyNoop(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// Apply empty config twice — second should be no-op.
	proposed := AgentConfig{}
	impact1, _ := n.SafeApply(context.Background(), agent.ID, proposed)
	impact2, _ := n.SafeApply(context.Background(), agent.ID, proposed)

	// First apply on empty agent: all fields are "changed" from zero value.
	// Second apply: no change.
	if len(impact2.ChangedFields) != 0 {
		t.Errorf("second identical apply should be no-op, got changed: %v", impact2.ChangedFields)
	}
	_ = impact1
}
