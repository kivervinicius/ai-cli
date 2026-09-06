# Nexus V1 — Agent Model

Per charter §19-24, §90-91. Persistent Agent is the primary operational unit.
Implemented in `internal/nexus/store` + `internal/nexus` service, served via
`/api/v1/agents*`.

## Model

```json
Agent {
  "id": "agt_01J…",             // stable across runtime/account/provider/terminal changes (§20)
  "project_id": "prj_…",        // every normal agent is project-scoped (§41)
  "name": "Backend Developer",
  "role": "",
  "status": "STOPPED",          // STARTING|WORKING|WAITING|APPROVAL|RATE_LIMITED|… (§40)
  "current_revision_id": "rev_…",
  "continuity_status": "LIVE_SAME_RUNTIME",  // (§24)
  "created_at": "…", "updated_at": "…", "last_started_at": "…"
}

AgentRevision { "id": "rev_…", "agent_id", "revision": 1, "config": "{}" }  // immutable, §23
RuntimeGeneration { "id": "gen_…", "agent_id", "revision_id", "runtime_id",
                    "provider", "profile", "provider_session", "continuity",
                    "started_at", "stopped_at", "state" }                   // §22
Lineage { "relation": "ACCOUNT_HANDOFF|CONTEXT_HANDOFF", source/target runtime+session, checkpoint_id }  // §39
```

## Continuity (honest, §24)

`LIVE_SAME_RUNTIME` · `REATTACHED_SAME_RUNTIME` · `NATIVE_RESUME_VERIFIED` ·
`NATIVE_RESUME_UNVERIFIED` · `CONTEXT_RECOVERED_NEW_SESSION` · `CONTINUITY_FAILED`.

Continuity is only set to VERIFIED when the resume is actually verified against
provider storage; the existing `ResumeVerifier` (prior gate) prevents blind
session-ID copying. Intent is never shown as fact (§86).

## Start flow (implemented)

`POST /api/v1/agents/:id/start` → `nexus.ResolveStartParams` → `nexus.StartAgent`:

### Resource allocation (prerequisite)
1. UI calls `startAgent` without provider/profile.
2. Backend returns `409 REQUIRED_RESOURCE_SELECTION` if no `AgentConfig` is persisted.
3. UI opens `ResourcePicker` → `GET /api/v1/resources` (real discovery).
4. User selects → `POST /api/v1/resources/select` → `AllocateResource` persists to `AgentConfig`.

### Launch
1. `ResolveStartParams(agentID, provider, profile)` — resolves: explicit > AgentConfig > error.
2. Load agent (project-scoped).
3. Read full `AgentConfig` from current revision (model, options, environment, isolation).
4. Add config revision (records full AgentConfig JSON).
5. `launcher.Launch` with full config propagation (provider, profile, workspace, model, environment, isolation, options).
6. Record `RuntimeGeneration` linking runtime + revision.
7. Agent → `WORKING`, `current_revision_id` set, continuity `LIVE_SAME_RUNTIME`.

### Config propagation
`LaunchOptions` now includes `Model`, `Environment`, `Isolation`, `Options` from `AgentConfig`.
Environment variables are injected into the runtime process environment.

Stop flow: current generation resolved → runtime stopped via protocol client →
generation marked STOPPED → agent → `STOPPED`.

## Agent terminal (charter §30-31, §91)

`GET /api/v1/agents/:id/terminal` resolves the current generation's `RuntimeID`,
registers with `AgentTerminalBroker`, and bridges to the existing authenticated
WebSocket terminal hub (xterm → SessionHost → PTY/ConPTY). Frontend keys the
terminal by AgentID so runtime changes never destroy the view.

The broker observes runtime generation changes via `WatchRuntimeChanged()` and
emits `runtime_changed` frames to connected browsers, enabling seamless
reconnection without terminal remount.

## API (charter §90)

| Method | Route | Status |
|---|---|---|
| GET/POST | `/api/v1/projects/:id/agents` | ✅ |
| GET/PATCH/DELETE | `/api/v1/agents/:id` | ✅ (GET includes generations, lineage, revisions) |
| POST | `/api/v1/agents/:id/start` | ✅ |
| POST | `/api/v1/agents/:id/stop` | ✅ |
| WS | `/api/v1/agents/:id/terminal` | ✅ (101 verified) |
| recover / reconfigure / handoff | pending Gate 3/6 | planned |

## Tests

- Store: agent CRUD, revisions increment, generations + current, lineage, **IDOR guard**
  (cross-project agent access → not found), delete cascade.
- Web: create/list/detail/layout, terminal 404 without runtime, CSRF.
- Live: create agent → start → runtime `fake-…` RUNNING → generation recorded → terminal 101.
