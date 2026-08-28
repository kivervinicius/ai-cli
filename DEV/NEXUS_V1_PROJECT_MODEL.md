# Nexus V1 — Project Model

Per charter §14-15, §89. Project is the root of the domain. Implemented in
`internal/nexus/store` (SQLite) and served via `/api/v1/projects*`.

## Model

```json
{
  "id": "prj_01J…",            // ULID-based, stable, never path-derived
  "name": "Omega API",
  "slug": "omega-api",          // derived from name, disambiguated on collision
  "canonical_path": "/home/user/company/api",  // Abs→Clean→EvalSymlinks, must be dir
  "repo_remote": "origin",
  "repo_url": "https://github.com/…",
  "default_branch": "main",
  "maestro_mode": "ASSIST",     // OFF | ASSIST | ORCHESTRATE (§52)
  "resource_policy": "{}",      // JSON; Project overrides Global (§48)
  "default_isolation": "developer",
  "settings": "{}",
  "created_at": "…",
  "updated_at": "…",
  "last_opened_at": "…"         // drives MRU ordering
}
```

## Identity rules

- ID is ULID (`prj_` prefix) — `/home/a/api` and `/home/b/api` are distinct projects with distinct IDs (charter §14).
- Canonical path is unique in the store; same directory cannot be registered twice.
- Path canonicalization rejects empty paths, non-existent paths, and files (must be a directory).

## Ordering

- `ListProjects` returns MRU-first (`last_opened_at DESC NULLS LAST`, then `created_at DESC`).
- `TouchProject` updates `last_opened_at` on open (deterministic MRU, no map iteration).

## API (charter §89)

| Method | Route | Status |
|---|---|---|
| GET | `/api/v1/projects` | ✅ |
| POST | `/api/v1/projects` `{name?, path}` | ✅ |
| GET | `/api/v1/projects/:id` | ✅ (project + layout) |
| PATCH | `/api/v1/projects/:id` | ✅ (name/mode/isolation/branch/policy) |
| DELETE | `/api/v1/projects/:id` | ✅ (cascades agents/revisions/generations) |
| GET/PUT | `/api/v1/projects/:id/layout` | ✅ |

## Tests

- Store: create (slug, mode default, ULID prefix), MRU ordering, delete cascade, canonical-path validation, layout round-trip.
- Web: POST/GET/PATCH/DELETE project, layout round-trip, CSRF enforced.
- Live: project created with `prj_…`, survives web server restart with same ID (A1).
