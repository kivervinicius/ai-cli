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
	return nil
}

// SchemaVersion returns the highest applied migration version.
func (s *Store) SchemaVersion() (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&v)
	return v, err
}
