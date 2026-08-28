# Nexus V1 — Final Engineering Report (Gate 1)

## Baseline & branch topology

- **Baseline:** `f9cd679` (validated production-readiness branch, on origin)
- **Integration branch:** `feat/nexus-v1` (this report's commit)
- **Maestro WIP audited:** `IAPro-Community/Orquestrador-Maestro@feat/cli-novo-wip` (salvage inventory, no merge)

## What was built (Gate 1 — Product Foundation)

### Backend
- `internal/nexus/store` — SQLite (modernc.org/sqlite, pure Go, no CGO) with embedded
  idempotent migrations; tables: projects, agents, agent_revisions, runtime_generations,
  lineage, events_metadata, maestro_advice, verification_evidence, project_layouts.
- Project domain: ULID id (`prj_…`), slug, canonical path (Abs→Clean→EvalSymlinks→dir),
  repo fields, maestro_mode (default ASSIST), resource policy, isolation, MRU ordering.
- Persistent Agent domain: stable `agt_…` id, project-scoped (IDOR guard), immutable
  config revisions, runtime generations, lineage, honest continuity statuses.
- `internal/nexus` service: `StartAgent`/`StopAgent` bridging store + RuntimeLauncher
  (records revision + generation, updates status).
- Web API: `/api/v1/projects*`, `/api/v1/agents*`, agent terminal WS
  (`/api/v1/agents/:id/terminal` → current generation → terminal hub).

### Frontend
- Design tokens (`:root` vars, dark primary, light/system prepared).
- UI primitives (Button, Badge, Card, Input, Spinner, EmptyState).
- AppShell (Nexus + Legacy nav) + CommandPalette (Ctrl/Cmd+K).
- ProjectsPage: project rail (add/select/delete), agents (create/start/stop), agent
  terminal frame. `App.tsx` slimmed (no god component).
- Nexus API client (CSRF-aware).

## Verification

- `go vet ./...` → clean
- `go test -race ./...` → **25 packages ok, 0 fail** (was 24)
- Frontend: typecheck, lint, unit (3), build → pass
- Live (Linux): project created (ULID), agent created, **start agent → runtime
  RUNNING**, generation recorded, **agent terminal WS 101**, **project survives
  web restart (same ID)** — the first vertical slice works.

## Gates status

| Gate | Scope | Status |
|---|---|---|
| 0 | baseline/archaeology/benchmark | ✅ |
| 1 | foundation: project + SQLite + design system + slice | ✅ (this commit) |
| 2 | persistent agent detail/recover UI | pending |
| 3 | configuration drawer + safe apply | pending |
| 4 | terminal broker + layout + replay | pending |
| 5 | Resource Scheduler | pending |
| 6 | Maestro Assist | pending |
| 7 | Mission Beta | pending |
| 8 | production polish + RC | pending |

## Known limitations (honest)

- Resource Scheduler, Maestro bridge, Mission Beta: not implemented (designed; later gates).
- `iapro` CLI rename + `iapro doctor`: Gate 8 (legacy `ai` commands preserved).
- Windows/macOS runtime E2E: pending CI + user-local (unchanged from prior gate).
- Session expiry/rotation + web split view: P2 backlog from prior gate.

## Verdict (this gate)

**CONDITIONAL_GO** for Gate 1: vertical slice functional and verified on Linux;
product-wide GO requires gates 2-8 + independent final review + Windows/macOS
runtime evidence (per §182-185).
