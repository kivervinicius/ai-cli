package nexus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// storeRunRepository adapts the product SQLite store to the runner's durable
// repository contract. Lease columns are authoritative; payload JSON carries
// the rest of the immutable/auditable mission state.
type storeRunRepository struct {
	st *store.Store
}

func newStoreRunRepository(st *store.Store) runner.RunRepository {
	return &storeRunRepository{st: st}
}

func (r *storeRunRepository) SaveRun(_ context.Context, run *runner.MissionRun) error {
	if r.st == nil {
		return fmt.Errorf("nexus store unavailable")
	}
	if run == nil || run.ID == "" {
		return fmt.Errorf("mission run is required")
	}
	payload, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal mission run: %w", err)
	}
	return r.st.UpsertMissionRun(store.MissionRunRecord{
		ID:             run.ID,
		PlanID:         run.PlanID,
		ProjectID:      run.ProjectID,
		State:          string(run.State),
		PayloadJSON:    string(payload),
		LeaseOwner:     run.LeaseOwner,
		LeaseToken:     run.LeaseToken,
		LeaseExpiresAt: run.LeaseExpiresAt,
		HeartbeatAt:    run.HeartbeatAt,
		CreatedAt:      run.StartedAt,
		UpdatedAt:      run.UpdatedAt,
	})
}

func (r *storeRunRepository) GetRun(_ context.Context, id string) (*runner.MissionRun, error) {
	rec, err := r.st.GetMissionRun(id)
	if err != nil {
		return nil, mapRunStoreError(err)
	}
	return decodeMissionRun(rec)
}

func (r *storeRunRepository) ListRuns(_ context.Context) ([]*runner.MissionRun, error) {
	recs, err := r.st.ListMissionRuns()
	if err != nil {
		return nil, err
	}
	out := make([]*runner.MissionRun, 0, len(recs))
	for i := range recs {
		run, err := decodeMissionRun(&recs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *storeRunRepository) AcquireLease(_ context.Context, id, owner string, ttl time.Duration) (*runner.MissionRun, error) {
	rec, err := r.st.AcquireMissionLease(id, owner, ttl)
	if err != nil {
		if isLeaseError(err) {
			return nil, runner.ErrLeaseHeld
		}
		return nil, mapRunStoreError(err)
	}
	return decodeMissionRun(rec)
}

func (r *storeRunRepository) RenewLease(_ context.Context, id, owner, token string, ttl time.Duration) error {
	if err := r.st.RenewMissionLease(id, owner, token, ttl); err != nil {
		if isLeaseError(err) {
			return runner.ErrLeaseHeld
		}
		return mapRunStoreError(err)
	}
	return nil
}

func (r *storeRunRepository) ReleaseLease(_ context.Context, id, owner, token string) error {
	if err := r.st.ReleaseMissionLease(id, owner, token); err != nil {
		if isLeaseError(err) {
			return runner.ErrLeaseHeld
		}
		return mapRunStoreError(err)
	}
	return nil
}

func decodeMissionRun(rec *store.MissionRunRecord) (*runner.MissionRun, error) {
	if rec == nil {
		return nil, runner.ErrRunNotFound
	}
	var run runner.MissionRun
	if err := json.Unmarshal([]byte(rec.PayloadJSON), &run); err != nil {
		return nil, fmt.Errorf("decode mission run %s: %w", rec.ID, err)
	}
	// Lease columns are written atomically and may be newer than payload_json.
	run.LeaseOwner = rec.LeaseOwner
	run.LeaseToken = rec.LeaseToken
	run.LeaseExpiresAt = rec.LeaseExpiresAt
	run.HeartbeatAt = rec.HeartbeatAt
	if rec.State != "" {
		run.State = runner.State(rec.State)
	}
	if !rec.UpdatedAt.IsZero() {
		run.UpdatedAt = rec.UpdatedAt
	}
	return &run, nil
}

func mapRunStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrNotFound) {
		return runner.ErrRunNotFound
	}
	return err
}

func isLeaseError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "mission run lease held" || msg == "mission run lease fencing mismatch"
}
