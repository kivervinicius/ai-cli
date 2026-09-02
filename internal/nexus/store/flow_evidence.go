package store

import (
	"database/sql"
	"errors"
	"time"
)

type FlowContextCapsuleRecord struct {
	ID           string
	RunID        string
	StepID       string
	FlowRevision int
	ContentJSON  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type FlowWorkReceiptRecord struct {
	ID          string
	RunID       string
	StepID      string
	Status      string
	ContentJSON string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Store) UpsertFlowContextCapsule(record FlowContextCapsuleRecord) (*FlowContextCapsuleRecord, error) {
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	_, err := s.db.Exec(`INSERT INTO flow_context_capsules(id, run_id, step_id, flow_revision, content_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id, step_id) DO UPDATE SET
			flow_revision=excluded.flow_revision,
			content_json=excluded.content_json,
			updated_at=excluded.updated_at`,
		record.ID, record.RunID, record.StepID, record.FlowRevision, record.ContentJSON,
		record.CreatedAt.Format(time.RFC3339Nano), record.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.GetFlowContextCapsule(record.RunID, record.StepID)
}

func (s *Store) GetFlowContextCapsule(runID, stepID string) (*FlowContextCapsuleRecord, error) {
	var record FlowContextCapsuleRecord
	var created, updated string
	err := s.db.QueryRow(`SELECT id, run_id, step_id, flow_revision, content_json, created_at, updated_at
		FROM flow_context_capsules WHERE run_id=? AND step_id=?`, runID, stepID).Scan(
		&record.ID, &record.RunID, &record.StepID, &record.FlowRevision, &record.ContentJSON, &created, &updated)
	if err != nil {
		return nil, err
	}
	record.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &record, nil
}

func (s *Store) UpsertFlowWorkReceipt(record FlowWorkReceiptRecord) (*FlowWorkReceiptRecord, error) {
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	_, err := s.db.Exec(`INSERT INTO flow_work_receipts(id, run_id, step_id, status, content_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id, step_id) DO UPDATE SET
			status=excluded.status,
			content_json=excluded.content_json,
			updated_at=excluded.updated_at`,
		record.ID, record.RunID, record.StepID, record.Status, record.ContentJSON,
		record.CreatedAt.Format(time.RFC3339Nano), record.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.GetFlowWorkReceipt(record.RunID, record.StepID)
}

func (s *Store) GetFlowWorkReceipt(runID, stepID string) (*FlowWorkReceiptRecord, error) {
	var record FlowWorkReceiptRecord
	var created, updated string
	err := s.db.QueryRow(`SELECT id, run_id, step_id, status, content_json, created_at, updated_at
		FROM flow_work_receipts WHERE run_id=? AND step_id=?`, runID, stepID).Scan(
		&record.ID, &record.RunID, &record.StepID, &record.Status, &record.ContentJSON, &created, &updated)
	if err != nil {
		return nil, err
	}
	record.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &record, nil
}

func (s *Store) ListFlowContextCapsules(runID string) ([]FlowContextCapsuleRecord, error) {
	rows, err := s.db.Query(`SELECT id, run_id, step_id, flow_revision, content_json, created_at, updated_at
		FROM flow_context_capsules WHERE run_id=? ORDER BY created_at, step_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FlowContextCapsuleRecord{}
	for rows.Next() {
		var record FlowContextCapsuleRecord
		var created, updated string
		if err := rows.Scan(&record.ID, &record.RunID, &record.StepID, &record.FlowRevision, &record.ContentJSON, &created, &updated); err != nil {
			return nil, err
		}
		record.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) ListFlowWorkReceipts(runID string) ([]FlowWorkReceiptRecord, error) {
	rows, err := s.db.Query(`SELECT id, run_id, step_id, status, content_json, created_at, updated_at
		FROM flow_work_receipts WHERE run_id=? ORDER BY created_at, step_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FlowWorkReceiptRecord{}
	for rows.Next() {
		var record FlowWorkReceiptRecord
		var created, updated string
		if err := rows.Scan(&record.ID, &record.RunID, &record.StepID, &record.Status, &record.ContentJSON, &created, &updated); err != nil {
			return nil, err
		}
		record.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, record)
	}
	return out, rows.Err()
}

func IsFlowEvidenceNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
