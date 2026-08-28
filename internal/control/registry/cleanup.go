package registry

import (
	"os"

	"github.com/kivervinicius/ai-cli/internal/control/protocol"
)

// CleanupStale scans registered active sessions, checks if their process/endpoint is dead,
// marks dead sessions as STALE, and removes orphaned socket files.
func (r *Registry) CleanupStale() (int, error) {
	r.mu.Lock()
	var candidates []RuntimeSession
	for _, s := range r.sessions {
		if s.IsActive() {
			candidates = append(candidates, s)
		}
	}
	r.mu.Unlock()

	cleaned := 0
	for _, s := range candidates {
		alive := IsProcessAliveWithGeneration(s.PID, s.HostGeneration)
		if !alive {
			// Check if endpoint is responsive
			client, err := protocol.NewClient(s.RuntimeID)
			if err == nil {
				_ = client.Close()
				continue
			}

			r.mu.Lock()
			current, ok := r.sessions[s.RuntimeID]
			if ok && current.IsActive() {
				// Mark as stale
				current.State = StateStale
				r.sessions[s.RuntimeID] = current
				cleaned++
			}
			r.mu.Unlock()

			// Clean up stale socket file if present
			sockPath := protocol.EndpointPath(s.RuntimeID)
			_ = os.Remove(sockPath)
		}
	}

	if cleaned > 0 {
		r.mu.Lock()
		_ = r.saveLocked()
		r.mu.Unlock()
	}

	return cleaned, nil
}

// PurgeInactive permanently removes STALE, FAILED, and STOPPED sessions whose PID and socket are dead.
func (r *Registry) PurgeInactive() (int, error) {
	r.mu.Lock()
	var candidates []RuntimeSession
	for _, s := range r.sessions {
		if !s.IsActive() {
			candidates = append(candidates, s)
		}
	}
	r.mu.Unlock()

	purged := 0
	for _, s := range candidates {
		alive := IsProcessAliveWithGeneration(s.PID, s.HostGeneration)
		if !alive {
			// Clean up socket file
			sockPath := protocol.EndpointPath(s.RuntimeID)
			_ = os.Remove(sockPath)

			r.mu.Lock()
			if _, ok := r.sessions[s.RuntimeID]; ok {
				delete(r.sessions, s.RuntimeID)
				purged++
			}
			r.mu.Unlock()
		}
	}

	if purged > 0 {
		r.mu.Lock()
		_ = r.saveLocked()
		r.mu.Unlock()
	}

	return purged, nil
}
