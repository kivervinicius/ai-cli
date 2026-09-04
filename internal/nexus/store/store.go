// Package store provides the durable Nexus product state backed by SQLite
// (pure-Go driver, no CGO, single portable binary).
//
// The SQLite store is the product domain store (projects, agents, revisions,
// runtime generations, lineage, layouts). Live low-level runtime state stays in
// internal/control/registry.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the Nexus durable product state handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the Nexus SQLite database at path and runs
// all pending schema migrations idempotently.
func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open nexus store: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping nexus store: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying handle for advanced queries.
func (s *Store) DB() *sql.DB { return s.db }

// migrate applies embedded migrations in filename order, recording each
// applied version in schema_migrations. Re-runs are no-ops.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, entry := range entries {
		name := filepath.Base(entry)
		var version int
		if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
			continue
		}
		applied := false
		_ = s.db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version=?`, version).Scan(&applied)
		_ = applied
		var exists int
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=?)`, version).Scan(&exists)
		if err != nil {
			return err
		}
		if exists == 1 {
			continue
		}

		body, err := migrationsFS.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	// Migration numbers alone are insufficient: databases can have a recorded
	// migration with a missing or incompatible table (for example after a
	// partial restore). Validate the structural contract on every open and
	// repair drift idempotently before exposing the store to callers.
	if err := s.repairFlowEvidenceSchema(); err != nil {
		return err
	}
	return nil
}

var flowEvidenceColumns = map[string][]string{
	"flow_context_capsules": {"id", "run_id", "step_id", "flow_revision", "content_json", "created_at", "updated_at"},
	"flow_work_receipts":    {"id", "run_id", "step_id", "status", "content_json", "created_at", "updated_at"},
}

func (s *Store) repairFlowEvidenceSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin schema repair: %w", err)
	}
	defer tx.Rollback()
	for table, required := range flowEvidenceColumns {
		cols, exists, err := tableColumns(tx, table)
		if err != nil {
			return err
		}
		if exists && hasColumns(cols, required) {
			continue
		}
		if exists {
			legacy := table + "_legacy_" + time.Now().UTC().Format("20060102150405")
			if _, err := tx.Exec(`ALTER TABLE ` + quoteIdent(table) + ` RENAME TO ` + quoteIdent(legacy)); err != nil {
				return fmt.Errorf("preserve legacy %s: %w", table, err)
			}
		}
		if err := createFlowEvidenceTable(tx, table); err != nil {
			return err
		}
		if exists && table == "flow_work_receipts" {
			// Best-effort copy from known legacy column names. Rows remain
			// auditable even when a legacy schema used payload_json/receipt_json.
			legacy := table + "_legacy_" + time.Now().UTC().Format("20060102150405")
			// Locate the just-renamed table (timestamp may differ by a second).
			legacy, err = latestLegacyTable(tx, table+"_legacy_")
			if err != nil {
				return err
			}
			lcols, _, _ := tableColumns(tx, legacy)
			if hasColumns(lcols, []string{"id", "run_id", "step_id"}) {
				content := "''"
				for _, c := range []string{"content_json", "payload_json", "receipt_json"} {
					if lcols[c] {
						content = quoteIdent(c)
						break
					}
				}
				status := "'UNKNOWN'"
				if lcols["status"] {
					status = quoteIdent("status")
				}
				created := "datetime('now')"
				if lcols["created_at"] {
					created = quoteIdent("created_at")
				}
				updated := created
				if lcols["updated_at"] {
					updated = quoteIdent("updated_at")
				}
				_, _ = tx.Exec(`INSERT OR IGNORE INTO flow_work_receipts(id,run_id,step_id,status,content_json,created_at,updated_at) SELECT id,run_id,step_id,` + status + `,` + content + `,` + created + `,` + updated + ` FROM ` + quoteIdent(legacy))
			}
		}
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_flow_context_capsules_run ON flow_context_capsules(run_id, step_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_flow_work_receipts_run ON flow_work_receipts(run_id, step_id)`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema repair: %w", err)
	}
	return nil
}

func tableColumns(tx *sql.Tx, table string) (map[string]bool, bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + quoteIdent(table) + `)`)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	var cid int
	var name, typ string
	var notnull, pk int
	var dflt any
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, false, err
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return cols, len(cols) > 0, nil
}

func hasColumns(cols map[string]bool, required []string) bool {
	for _, c := range required {
		if !cols[c] {
			return false
		}
	}
	return true
}

func tableContractValid(tx *sql.Tx, table string) bool {
	rows, err := tx.Query(`PRAGMA foreign_key_list(` + quoteIdent(table) + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	var id, seq int
	var tbl, from, to, onUpdate, onDelete, match sql.NullString
	hasRunFK := false
	for rows.Next() {
		if rows.Scan(&id, &seq, &tbl, &from, &to, &onUpdate, &onDelete, &match) == nil && tbl.String == "mission_runs" && from.String == "run_id" {
			hasRunFK = true
		}
	}
	if !hasRunFK {
		return false
	}
	idxRows, err := tx.Query(`PRAGMA index_list(` + quoteIdent(table) + `)`)
	if err != nil {
		return false
	}
	defer idxRows.Close()
	var seqNo int
	var name string
	var unique, origin, partial int
	for idxRows.Next() {
		if idxRows.Scan(&seqNo, &name, &unique, &origin, &partial) != nil || unique == 0 {
			continue
		}
		info, e := tx.Query(`PRAGMA index_info(` + quoteIdent(name) + `)`)
		if e != nil {
			continue
		}
		seen := map[string]bool{}
		var s, cid int
		var col string
		for info.Next() {
			if info.Scan(&s, &cid, &col) == nil {
				seen[col] = true
			}
		}
		info.Close()
		if seen["run_id"] && seen["step_id"] {
			return true
		}
	}
	return false
}
func quoteIdent(v string) string { return `"` + strings.ReplaceAll(v, `"`, `""`) + `"` }
func createFlowEvidenceTable(tx *sql.Tx, table string) error {
	stmt := `CREATE TABLE IF NOT EXISTS ` + quoteIdent(table) + ` (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE, step_id TEXT NOT NULL, `
	if table == "flow_context_capsules" {
		stmt += `flow_revision INTEGER NOT NULL, `
	} else {
		stmt += `status TEXT NOT NULL, `
	}
	stmt += `content_json TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(run_id, step_id))`
	_, err := tx.Exec(stmt)
	return err
}
func latestLegacyTable(tx *sql.Tx, prefix string) (string, error) {
	var name string
	err := tx.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE ? ORDER BY name DESC LIMIT 1`, prefix+"%").Scan(&name)
	return name, err
}

// SchemaVersion returns the highest applied migration version.
func (s *Store) SchemaVersion() (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&v)
	return v, err
}
