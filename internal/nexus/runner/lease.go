package runner

import (
	"sync"
	"time"
)

// LeaseManager coordinates runtime leases and heartbeats for active mission runs (§Phase F).
type LeaseManager struct {
	mu         sync.RWMutex
	heartbeats map[string]time.Time // runID -> last heartbeat
	activeRuns map[string]*MissionRun
}

func NewLeaseManager() *LeaseManager {
	return &LeaseManager{
		heartbeats: make(map[string]time.Time),
		activeRuns: make(map[string]*MissionRun),
	}
}

func (m *LeaseManager) RegisterRun(run *MissionRun) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRuns[run.ID] = run
	m.heartbeats[run.ID] = time.Now().UTC()
}

func (m *LeaseManager) Heartbeat(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats[runID] = time.Now().UTC()
}

func (m *LeaseManager) UnregisterRun(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeRuns, runID)
	delete(m.heartbeats, runID)
}

func (m *LeaseManager) GetRun(runID string) (*MissionRun, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	run, ok := m.activeRuns[runID]
	return run, ok
}

func (m *LeaseManager) ListActiveRuns() []*MissionRun {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*MissionRun, 0, len(m.activeRuns))
	for _, r := range m.activeRuns {
		list = append(list, r)
	}
	return list
}
