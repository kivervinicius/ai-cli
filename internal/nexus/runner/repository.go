package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrRunNotFound = errors.New("mission run not found")
var ErrLeaseHeld = errors.New("mission run lease held by another worker")

// RunRepository is the durable authority for MissionRun state and worker leases.
type RunRepository interface {
	SaveRun(context.Context, *MissionRun) error
	GetRun(context.Context, string) (*MissionRun, error)
	ListRuns(context.Context) ([]*MissionRun, error)
	AcquireLease(context.Context, string, string, time.Duration) (*MissionRun, error)
	RenewLease(context.Context, string, string, string, time.Duration) error
	ReleaseLease(context.Context, string, string, string) error
}

// MemoryRunRepository is test/dev only. Production Nexus injects the SQLite-backed adapter.
type MemoryRunRepository struct {
	mu   sync.Mutex
	runs map[string]*MissionRun
}

func NewMemoryRunRepository() *MemoryRunRepository {
	return &MemoryRunRepository{runs: map[string]*MissionRun{}}
}

func cloneRun(in *MissionRun) *MissionRun {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out MissionRun
	_ = json.Unmarshal(b, &out)
	return &out
}
func (m *MemoryRunRepository) SaveRun(_ context.Context, run *MissionRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run == nil || run.ID == "" {
		return fmt.Errorf("mission run is required")
	}
	if current, ok := m.runs[run.ID]; ok {
		if current.LeaseToken != "" && current.LeaseToken != run.LeaseToken {
			return ErrLeaseHeld
		}
		copy := cloneRun(run)
		// Lease metadata is maintained only by Acquire/Renew/Release and must
		// never move backwards because a worker saved an older in-memory copy.
		copy.LeaseOwner = current.LeaseOwner
		copy.LeaseToken = current.LeaseToken
		copy.LeaseExpiresAt = current.LeaseExpiresAt
		copy.HeartbeatAt = current.HeartbeatAt
		m.runs[run.ID] = copy
		return nil
	}
	m.runs[run.ID] = cloneRun(run)
	return nil
}
func (m *MemoryRunRepository) GetRun(_ context.Context, id string) (*MissionRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return nil, ErrRunNotFound
	}
	return cloneRun(r), nil
}
func (m *MemoryRunRepository) ListRuns(_ context.Context) ([]*MissionRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*MissionRun, 0, len(m.runs))
	for _, r := range m.runs {
		out = append(out, cloneRun(r))
	}
	return out, nil
}
func (m *MemoryRunRepository) AcquireLease(_ context.Context, id, owner string, ttl time.Duration) (*MissionRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return nil, ErrRunNotFound
	}
	now := time.Now().UTC()
	if r.LeaseOwner != "" && r.LeaseOwner != owner && r.LeaseExpiresAt != nil && r.LeaseExpiresAt.After(now) {
		return nil, ErrLeaseHeld
	}
	token := fmt.Sprintf("lease-%s-%d", owner, now.UnixNano())
	expires := now.Add(ttl)
	r.LeaseOwner, r.LeaseToken, r.LeaseExpiresAt, r.HeartbeatAt = owner, token, &expires, &now
	return cloneRun(r), nil
}
func (m *MemoryRunRepository) RenewLease(_ context.Context, id, owner, token string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	if r.LeaseOwner != owner || r.LeaseToken != token {
		return ErrLeaseHeld
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	r.HeartbeatAt, r.LeaseExpiresAt = &now, &expires
	return nil
}
func (m *MemoryRunRepository) ReleaseLease(_ context.Context, id, owner, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	if r.LeaseOwner != owner || r.LeaseToken != token {
		return ErrLeaseHeld
	}
	r.LeaseOwner, r.LeaseToken, r.LeaseExpiresAt = "", "", nil
	return nil
}
