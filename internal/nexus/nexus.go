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

	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
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
