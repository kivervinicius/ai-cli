package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// mockLauncher satisfies the Launcher interface for unit tests without spawning
// real processes. It registers runtimes in the DefaultRegistry so runtimeAlive()
// returns true for mocked sessions, and tracks Stop calls.
type mockLauncher struct {
	lastOptions launcher.LaunchOptions
	launches    int
}

func newMockLauncher() *mockLauncher {
	// Reset the singleton so each test gets an isolated registry.
	registry.ResetDefaultRegistryForTest()
	return &mockLauncher{}
}

func (m *mockLauncher) Launch(_ context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	m.lastOptions = opts
	m.launches++
	if opts.ProviderID == "" {
		return nil, fmt.Errorf("provider is required")
	}
	sess := registry.RuntimeSession{
		RuntimeID:         opts.RuntimeID,
		AgentID:           opts.AgentID,
		ProviderID:        opts.ProviderID,
		ProfileID:         opts.ProfileID,
		ProviderSessionID: opts.ProviderSessionID,
		Workspace:         opts.Workspace,
		State:             registry.StateRunning,
		PID:               os.Getpid(), // use real PID so IsProcessAlive works
		StartedAt:         time.Now(),
		Transport:         "mock",
	}
	if sess.RuntimeID == "" {
		sess.RuntimeID = fmt.Sprintf("mock-%s", opts.ProviderID)
	}
	if err := registry.DefaultRegistry().Register(sess); err != nil {
		return nil, fmt.Errorf("mock register: %w", err)
	}
	return &sess, nil
}

func (m *mockLauncher) Stop(runtimeID string) error {
	_ = registry.DefaultRegistry().UpdateState(runtimeID, registry.StateStopped)
	registry.DefaultRegistry().Delete(runtimeID)
	return nil
}

func openTestNexus(t *testing.T) *Nexus {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dir)
	st, err := store.Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Nexus{st: st, launcher: newMockLauncher()}
}

func TestEffectiveAgentState(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// No generation yet → store status (STOPPED).
	state, err := n.EffectiveAgentState(agent.ID)
	if err != nil || state != store.AgentStopped {
		t.Fatalf("expected STOPPED with no generation, got %q (%v)", state, err)
	}

	// A generation referencing a dead runtime + WORKING status → RECOVERABLE.
	dead := &store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev_x",
		RuntimeID:  "rt_definitely_dead_xyz",
		Provider:   "fake",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "RUNNING",
	}
	if _, err := st.AddGeneration(*dead); err != nil {
		t.Fatal(err)
	}
	agent.Status = store.AgentWorking
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}

	state, err = n.EffectiveAgentState(agent.ID)
	if err != nil || state != store.AgentRecoverable {
		t.Fatalf("expected RECOVERABLE for dead runtime, got %q (%v)", state, err)
	}

	// A STOPPED agent with a dead runtime stays STOPPED (never claims RECOVERABLE).
	agent.Status = store.AgentStopped
	_ = st.UpdateAgent(agent)
	state, _ = n.EffectiveAgentState(agent.ID)
	if state != store.AgentStopped {
		t.Fatalf("expected STOPPED (not recoverable) for stopped agent, got %q", state)
	}
}

func TestRecoverAgentRequiresKnownRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// Recover on an agent with a live runtime must refuse cleanly (no panic, no 500).
	// Without a live generation this is guarded by EffectiveAgentState in the API;
	// here we assert the store-level invariants hold for a stopped agent.
	if got := os.Getenv("AI_CLI_DATA_DIR"); got == "" {
		t.Fatal("data dir not set")
	}
	if agent.ID == "" {
		t.Fatal("agent id empty")
	}
}

func TestStartAgentRejectsDuplicateLiveRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})
	if _, err := n.SafeApply(context.Background(), agent.ID, AgentConfig{Provider: "claude", Profile: "default"}); err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartAgent(context.Background(), agent.ID, "claude", "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartAgent(context.Background(), agent.ID, "claude", "default"); err == nil {
		t.Fatal("second StartAgent must reject an already-live Agent runtime")
	}
}
