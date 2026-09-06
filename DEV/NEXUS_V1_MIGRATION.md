# Nexus V1 — Migration

## Approach

- Driver: `modernc.org/sqlite` (pure Go, no CGO → portable single binary, charter §63).
- DSN pragmas: `journal_mode(WAL)`, `foreign_keys(1)`, `busy_timeout(5000)`.
- Embedded migrations (`internal/nexus/store/migrations/*.sql`), applied in filename order,
  each in a transaction, recorded in `schema_migrations`. Idempotent on re-open
  (verified by test: reopen keeps same `MAX(version)`).

## What is stored (charter §64)

projects · agents · agent_revisions · runtime_generations · lineage ·
events_metadata (kind/ts/summary) · maestro_advice · verification_evidence ·
project_layouts · schema_migrations

## What is NEVER stored (charter §65)

API keys · OAuth tokens · auth.json · cookies · provider secrets · private keys ·
raw full terminal transcripts. `Agent.Env/Args/Binary` are already `json:"-"` in the
runtime registry; SQLite stores launch metadata only (provider, profile, session ID —
no secrets). Redaction pipeline is applied to checkpoints/events/reports.

## Legacy data

- Existing `projects.json` (workspace store) and registry `runtimes.json` remain the
  live low-level sources. SQLite is additive; a future backfill job can migrate
  workspace store entries into `projects` (idempotent, keeps `canonical_path` unique).
- No migration of live registry state into SQLite in this gate — runtime state stays
  in the registry (§66), avoiding duplication and data-loss risk.

## Safety

- Backup: database file is under `AI_CLI_DATA_DIR/nexus.db`; migration failure rolls
  back the transaction and surfaces an error (store unavailable → runtime features
  degrade gracefully instead of crashing — `nexus.OpenProject` returns a typed error).
