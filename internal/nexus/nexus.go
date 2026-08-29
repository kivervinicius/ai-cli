// Package nexus wires the durable product state (SQLite) to the control plane:
// projects, persistent agents, runtime generations, lineage, layouts.
package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// Nexus is the product-level service bridging the durable store and the
// control plane runtime layer.
type Nexus struct {
	mu       sync.Mutex
	st       *store.Store
	launcher *launcher.Launcher
}

var (
	defaultNexus *Nexus
	nexusOnce    sync.Once
)

// Default returns the process-wide Nexus service (SQLite store under DataDir).
func Default() *Nexus {
	nexusOnce.Do(func() {
		st, err := OpenStore()
		if err != nil {
			// The web layer surfaces errors per-request; keep a nil store so
			// runtime features degrade gracefully instead of panicking.
			st = nil
		}
		defaultNexus = &Nexus{st: st, launcher: launcher.Default()}
	})
	return defaultNexus
}

// OpenStore opens the SQLite store at <DataDir>/nexus.db.
func OpenStore() (*store.Store, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(dir, 0700)
	return store.Open(filepath.Join(dir, "nexus.db"))
}

// Store returns the underlying store (nil when unavailable).
func (n *Nexus) Store() *store.Store { return n.st }

// OpenProject returns the store or an error when the DB is unavailable.
func (n *Nexus) OpenProject() (*store.Store, error) {
	if n.st == nil {
		return nil, fmt.Errorf("nexus store unavailable (MAESTRO_DEGRADED-style degraded mode): sqlite open failed")
	}
	return n.st, nil
}

// StartAgent launches a supervised runtime for an agent, records a config
// revision and a runtime generation, and updates agent status.
func (n *Nexus) StartAgent(ctx context.Context, agentID, provider, profile string) (*registry.RuntimeSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}
	if provider == "" {
		provider = "fake" // default structural provider for the first vertical slice
	}
	if profile == "" {
		profile = "default"
	}

	rev, err := st.AddRevision(agentID, store.MustJSON(map[string]string{
		"provider": provider,
		"profile":  profile,
	}))
	if err != nil {
		return nil, err
	}

	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		ProviderID: provider,
		ProfileID:  profile,
	})
	if err != nil {
		return nil, fmt.Errorf("start agent runtime: %w", err)
	}

	gen := store.RuntimeGeneration{
		AgentID:         agentID,
		RevisionID:      rev.ID,
		RuntimeID:       sess.RuntimeID,
		Provider:        provider,
		Profile:         profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      store.ContinuityLiveSameRuntime,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(gen); err != nil {
		return nil, err
	}

	agent.Status = "WORKING"
	agent.CurrentRevisionID = rev.ID
	agent.ContinuityStatus = store.ContinuityLiveSameRuntime
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)
	return sess, nil
}

// StopAgent stops the agent's current runtime generation.
func (n *Nexus) StopAgent(ctx context.Context, agentID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return err
	}
	gen, err := st.CurrentGeneration(agentID)
	if err == nil && gen.RuntimeID != "" {
		n.stopRuntime(gen.RuntimeID)
		stopped := time.Now().UTC()
		_ = st.StopGeneration(gen.ID, stopped)
	}
	agent.Status = "STOPPED"
	_ = st.UpdateAgent(agent)
	return nil
}

func (n *Nexus) stopRuntime(runtimeID string) {
	client, err := protocol.NewClient(runtimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
}

// runtimeAlive reports whether a runtime's process is currently alive,
// consulting the live runtime registry (PID + host generation protection).
func (n *Nexus) runtimeAlive(runtimeID string) bool {
	if runtimeID == "" {
		return false
	}
	sess, ok := registry.DefaultRegistry().Get(runtimeID)
	if !ok {
		return false
	}
	return registry.IsProcessAlive(sess.PID)
}

// EffectiveAgentState derives the honest, live agent state: an agent whose
// runtime process is gone is RECOVERABLE (the Agent persists, the Runtime does
// not — §29). It never guesses; it reflects store status + live registry.
func (n *Nexus) EffectiveAgentState(agentID string) (string, error) {
	st, err := n.OpenProject()
	if err != nil {
		return "", err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return "", err
	}
	gen, gerr := st.CurrentGeneration(agentID)
	if gerr != nil {
		return agent.Status, nil // no runtime generation yet
	}
	if n.runtimeAlive(gen.RuntimeID) {
		return agent.Status, nil
	}
	switch agent.Status {
	case store.AgentWorking, store.AgentStarting, store.AgentHandoff, store.AgentRecovering:
		return store.AgentRecoverable, nil
	default:
		return agent.Status, nil
	}
}

// RecoverAgent restarts a persistent agent whose runtime died (machine reboot,
// host crash). It either performs provider-native resume (honest continuity:
// NATIVE_RESUME_UNVERIFIED, since provider-level verification needs a
// CONTROL_API adapter) or starts a NEW SESSION (CONTEXT_RECOVERED_NEW_SESSION)
// when no provider session is known.
func (n *Nexus) RecoverAgent(ctx context.Context, agentID string) (*registry.RuntimeSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}

	gen, gerr := st.CurrentGeneration(agentID)
	if gerr == nil && n.runtimeAlive(gen.RuntimeID) {
		return nil, fmt.Errorf("agent runtime is already alive (no recovery needed)")
	}

	provider := "fake"
	profile := "default"
	sessionID := ""
	if gerr == nil {
		provider = gen.Provider
		profile = gen.Profile
		sessionID = gen.ProviderSession
	}

	agent.Status = store.AgentRecovering
	_ = st.UpdateAgent(agent)

	rev, err := st.AddRevision(agentID, store.MustJSON(map[string]string{
		"provider": provider,
		"profile":  profile,
	}))
	if err != nil {
		return nil, err
	}

	var args []string
	continuity := store.ContinuityContextRecovered // NEW SESSION unless native resume is possible
	if sessionID != "" {
		if d, derr := driver.DefaultRegistry().Get(provider); derr == nil {
			prof := model.Profile{Name: profile, Provider: provider}
			if can, _ := d.CanResume(ctx, prof, sessionID); can {
				if ra, rerr := d.BuildResumeArgs(ctx, prof, sessionID); rerr == nil {
					args = ra
					// Honest: we resumed with the provider session, but cannot
					// claim VERIFIED without provider-side discovery (§28).
					continuity = store.ContinuityNativeResumeUnverified
				}
			}
		}
	}

	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		ProviderID:        provider,
		ProfileID:         profile,
		ProviderSessionID: sessionID,
		Args:              args,
	})
	if err != nil {
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		return nil, fmt.Errorf("recover agent runtime: %w", err)
	}

	gen2 := store.RuntimeGeneration{
		AgentID:         agentID,
		RevisionID:      rev.ID,
		RuntimeID:       sess.RuntimeID,
		Provider:        provider,
		Profile:         profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      continuity,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(gen2); err != nil {
		return nil, err
	}

	agent.Status = store.AgentWorking
	agent.CurrentRevisionID = rev.ID
	agent.ContinuityStatus = continuity
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)
	return sess, nil
}

func dataDir() (string, error) {
	if dir := os.Getenv("AI_CLI_DATA_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "ai-manager"), nil
}
