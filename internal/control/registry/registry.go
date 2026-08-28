package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
)

// Registry manages in-memory and persistent state of AI Control runtimes.
type Registry struct {
	mu       sync.RWMutex
	filePath string
	sessions map[string]RuntimeSession
}

var (
	defaultRegistry *Registry
	regOnce         sync.Once
)

// DefaultRegistry returns the singleton registry instance.
func DefaultRegistry() *Registry {
	regOnce.Do(func() {
		dataDir, err := config.DataDir()
		if err != nil {
			dataDir = filepath.Join(os.TempDir(), "ai-control")
		}
		path := filepath.Join(dataDir, "runtimes.json")
		defaultRegistry = NewRegistry(path)
	})
	return defaultRegistry
}

// NewRegistry initializes a Registry storing state at the given file path.
func NewRegistry(filePath string) *Registry {
	r := &Registry{
		filePath: filePath,
		sessions: make(map[string]RuntimeSession),
	}
	_ = r.load()
	return r
}

func (r *Registry) load() error {
	if r.filePath == "" {
		return nil
	}
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var list []RuntimeSession
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, s := range list {
		r.sessions[s.RuntimeID] = s
	}
	return nil
}

func (r *Registry) saveLocked() error {
	if r.filePath == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(r.filePath), 0700)

	var list []RuntimeSession
	for _, s := range r.sessions {
		list = append(list, s)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", r.filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}

	return os.Rename(tmp, r.filePath)
}

// Register adds or updates a runtime session.
func (r *Registry) Register(s RuntimeSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s.RuntimeID == "" {
		return errors.New("runtime ID cannot be empty")
	}
	now := time.Now()
	if s.StartedAt.IsZero() {
		s.StartedAt = now
	}
	s.UpdatedAt = now

	r.sessions[s.RuntimeID] = s
	return r.saveLocked()
}

// Get retrieves a runtime session by ID.
func (r *Registry) Get(runtimeID string) (RuntimeSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[runtimeID]
	return s, ok
}

// UpdateState updates the lifecycle state of a runtime.
func (r *Registry) UpdateState(runtimeID string, state RuntimeState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[runtimeID]
	if !ok {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	s.State = state
	s.UpdatedAt = time.Now()
	r.sessions[runtimeID] = s
	return r.saveLocked()
}

// UpdateProviderSessionID updates the underlying provider session ID.
func (r *Registry) UpdateProviderSessionID(runtimeID, providerSessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[runtimeID]
	if !ok {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	s.ProviderSessionID = providerSessionID
	s.UpdatedAt = time.Now()
	r.sessions[runtimeID] = s
	return r.saveLocked()
}

// List returns all registered runtime sessions.
func (r *Registry) List() []RuntimeSession {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []RuntimeSession{}
	for _, s := range r.sessions {
		result = append(result, s)
	}
	return result
}

// ListActive returns all active (running, starting, detached) sessions.
func (r *Registry) ListActive() []RuntimeSession {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []RuntimeSession{}
	for _, s := range r.sessions {
		if s.IsActive() {
			result = append(result, s)
		}
	}
	return result
}

// Delete removes a runtime from the registry.
func (r *Registry) Delete(runtimeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, runtimeID)
	return r.saveLocked()
}
