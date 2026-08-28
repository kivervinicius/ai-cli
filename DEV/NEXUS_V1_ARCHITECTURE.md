# Nexus V1 — Architecture

## Layers

```text
Web UI (React 19, xterm.js)  ──  Nexus shell, Projects → Agents → Terminal
        │  REST + WS (authenticated, CSRF, Origin)
Nexus API  internal/control/web  (projects/agents/runtimes/terminal)
        │
Nexus Service  internal/nexus    (orchestration: start/stop agent → runtime)
        │
Store (SQLite, pure Go)  internal/nexus/store   ← durable product state
        │
Control Core  internal/control   (registry, host, launcher, protocol, terminal)
        │
Providers: codex · claude · gemini · opencode · agy · cursor · fake
```

## Layering rules (charter §191, §190)

- **Project-first:** everything operational belongs to a `Project` (id ULID, never path-derived).
- **Persistent Agent:** `Agent` is stable (`agt_…`); runtimes are temporary `RuntimeGeneration` incarnations of it.
- **Web is the official UI;** TUI being phased out (kept as legacy for compatibility).
- **Maestro boundary:** Nexus consumes Maestro (methods/process/risk/gates); Maestro never depends on Nexus. No Maestro knowledge is duplicated into Nexus.
- **Shared Core:** the Web is a client of the same control core (single WriterLease, same SessionHost, same registry).
- **Honesty:** intent is never rendered as operational fact (`STOPPING` until confirmed); continuity statuses are honest (`NATIVE_RESUME_UNVERIFIED` when unverifiable).

## Data model

- `Project` → `Agent` (1:N) → `AgentRevision` (immutable, 1:N) and `RuntimeGeneration` (1:N)
- `RuntimeGeneration` references a concrete `RuntimeID`, provider/profile, provider session, continuity state.
- `Lineage` records Account/Context handoff edges per agent.
- `project_layouts` persists the per-project cockpit (open agents, splits).
- Live runtime state stays in `internal/control/registry`; SQLite is the durable product store (charter §66).

## Concurrency / correctness

- Writer lease lives in the Control Core (SessionHost); the Web/agent terminal broker is a client, not a second system (§92).
- Agent terminal WS resolves current generation → SessionHost → PTY/ConPTY; xterm is keyed by AgentID, never destroyed on runtime change (§30-31).
- Reconnect: bounded ring-buffer replay + live stream; browser close never stops the provider (§25).

## Security posture

- Loopback-only bind; public refused; private requires `--remote`; CGNAT-aware (existing).
- Bootstrap token → HttpOnly SameSite=Strict cookie; CSRF on state-changing REST; Origin validation on REST+WS; restrictive CSP; no wildcard CORS.
- Project path canonicalization (Abs → Clean → EvalSymlinks → must-exist-dir); Agent access is project-scoped (IDOR guard in store).
- Redaction pipeline applied to checkpoints/events/reports; secrets never stored in SQLite.

## Storage migration

- Embedded SQL migrations, recorded in `schema_migrations`, idempotent, transactional per migration. Current version: 1.
