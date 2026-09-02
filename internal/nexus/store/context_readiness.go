package store

import (
	"database/sql"
	"errors"
	"time"
)

type ContextReadinessRecord struct {
	ProjectID       string
	State           string
	FingerprintHash string
	FingerprintJSON string
	MaestroVersion  string
	Error           string
	HydratedAt      *time.Time
	UpdatedAt       time.Time
}

func (s *Store) GetContextReadiness(projectID string) (*ContextReadinessRecord, error) {
	var record ContextReadinessRecord
	var hydrated sql.NullString
	var updated string
	err := s.db.QueryRow(`SELECT project_id,state,fingerprint_hash,fingerprint_json,maestro_version,error,hydrated_at,updated_at FROM project_context_readiness WHERE project_id=?`, projectID).Scan(
		&record.ProjectID, &record.State, &record.FingerprintHash, &record.FingerprintJSON, &record.MaestroVersion, &record.Error, &hydrated, &updated)
	if err != nil {
		return nil, err
	}
	if hydrated.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, hydrated.String); parseErr == nil {
			record.HydratedAt = &parsed
		}
	}
	record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &record, nil
}

func (s *Store) PutContextReadiness(record ContextReadinessRecord) (*ContextReadinessRecord, error) {
	now := time.Now().UTC()
	record.UpdatedAt = now
	var hydrated any
	if record.HydratedAt != nil {
		hydrated = record.HydratedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.Exec(`INSERT INTO project_context_readiness(project_id,state,fingerprint_hash,fingerprint_json,maestro_version,error,hydrated_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(project_id) DO UPDATE SET state=excluded.state,fingerprint_hash=excluded.fingerprint_hash,fingerprint_json=excluded.fingerprint_json,maestro_version=excluded.maestro_version,error=excluded.error,hydrated_at=excluded.hydrated_at,updated_at=excluded.updated_at`,
		record.ProjectID, record.State, record.FingerprintHash, record.FingerprintJSON, record.MaestroVersion, record.Error, hydrated, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func IsContextReadinessMissing(err error) bool { return errors.Is(err, sql.ErrNoRows) }
