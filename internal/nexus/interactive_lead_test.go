package nexus

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestInteractiveLeadPersistsIdentityAndRuntimeGeneration(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Interactive lead", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	lead, err := n.PrepareInteractiveLead(context.Background(), project.ID, "agy", "work")
	if err != nil {
		t.Fatal(err)
	}
	if lead.Role != interactiveLeadRole || lead.CurrentRevisionID == "" {
		t.Fatalf("lead identity was not persisted: %+v", lead)
	}
	revision, err := st.GetRevision(lead.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseAgentConfig(revision.Config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "agy" || cfg.Profile != "work" || cfg.AgentSpec.Role != interactiveLeadRole {
		t.Fatalf("lead runtime preference/spec mismatch: %+v", cfg)
	}

	session := &registry.RuntimeSession{RuntimeID: "interactive-lead-runtime", AgentID: lead.ID, ProviderID: "agy", ProfileID: "work"}
	if err := n.BindInteractiveLeadRuntime(context.Background(), lead.ID, session); err != nil {
		t.Fatal(err)
	}
	registry.DefaultRegistry().Delete(session.RuntimeID)
	recoveredLead, err := n.PrepareInteractiveLead(context.Background(), project.ID, "codex", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if recoveredLead.ID != lead.ID {
		t.Fatalf("dead runtime must recover the existing Lead identity: got %s, want %s", recoveredLead.ID, lead.ID)
	}
	working, err := st.GetAgent(lead.ID, project.ID)
	if err != nil || working.Status != store.AgentWorking {
		t.Fatalf("lead must be WORKING while attached: %+v, err=%v", working, err)
	}
	if err := n.ReleaseInteractiveLeadRuntime(context.Background(), lead.ID, session.RuntimeID); err != nil {
		t.Fatal(err)
	}
	stopped, err := st.GetAgent(lead.ID, project.ID)
	if err != nil || stopped.Status != store.AgentStopped {
		t.Fatalf("lead must be STOPPED after detach: %+v, err=%v", stopped, err)
	}
	generation, err := st.GenerationByRuntimeID(session.RuntimeID)
	if err != nil || generation.StoppedAt == nil {
		t.Fatalf("lead generation must be closed after detach: %+v, err=%v", generation, err)
	}
}
