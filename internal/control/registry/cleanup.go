package registry

import (
	"os"
	"syscall"

	"github.com/kivervinicius/ai-cli/internal/control/protocol"
)

// IsProcessAlive checks whether a process with the given PID is currently running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, signal 0 tests process existence
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// CleanupStale scans registered active sessions, checks if their process/endpoint is dead,
// marks dead sessions as STALE, and removes orphaned socket files.
func (r *Registry) CleanupStale() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cleaned := 0
	for id, s := range r.sessions {
		if !s.IsActive() {
			continue
		}

		alive := IsProcessAlive(s.PID)
		if !alive {
			// Check if endpoint is responsive
			client, err := protocol.NewClient(s.RuntimeID)
			if err == nil {
				_ = client.Close()
				continue
			}

			// Mark as stale
			s.State = StateStale
			r.sessions[id] = s
			cleaned++

			// Clean up stale socket file if present
			sockPath := protocol.EndpointPath(s.RuntimeID)
			_ = os.Remove(sockPath)
		}
	}

	if cleaned > 0 {
		_ = r.saveLocked()
	}

	return cleaned, nil
}
