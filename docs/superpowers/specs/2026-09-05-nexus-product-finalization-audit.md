# IAPro Nexus Product Finalization — Audit Baseline

**Base branch:** `feat/nexus-maximum-delivery`
**Base SHA:** `00539ebdbcbed0478fcbd3c2a8a916410429d55`
**Campaign branch:** `feat/nexus-product-finalization`
**Capacity branch:** `feat/capacity-monitor-notifications` exists separately and was not merged.

## Evidence status

This document records the initial audit. A capability is not marked VERIFIED solely because a type, endpoint, component, test, or report exists.

| Domain | Status | Evidence summary | Missing proof / gap |
|---|---|---|---|
| Direct Mode | PARTIAL | Direct session surface and API exist; unit coverage passes. | No browser E2E proving Project → New AI Session → provider → terminal without Composer/Flow/Mission. |
| Persistent Agents | VERIFIED (L1/L2/L5) / PARTIAL (L3/L4) | Stable IDs, generations, lineage and recovery tests exist. | Real cross-restart/provider continuity and truthful status labels require E2E. |
| Mission Runner | VERIFIED (L1/L2/L5) / PARTIAL (L3/L4) | Durable states, retries, fencing and evidence tests pass. | Real provider/worktree restart run is not tested. |
| Scheduler | REGRESSED | Scheduling modes and scoring exist. | Schedule claim is not atomic before run creation; autonomous default can use project checkout. |
| Worktree isolation | REGRESSED | Explicit worktree mode works. | Autonomous execution defaults to `project`, violating fail-closed isolation. |
| Autonomy contract | PARTIAL | Contract types and snapshots exist. | Enforcement is prompt/path validation, not OS-level isolation; path rules are checked after execution. |
| Composer/readiness | PARTIAL | Persistence, readiness checks, unknown resolution and artifact versions exist. | Blocking gaps can be bypassed with `confirmGaps`; unknown status is not validated; frontend/backend readiness shapes diverge. |
| Maestro | PARTIAL | Catalog/status validation exists in Composer. | Decomposition/materialization/approved run paths bypass real Maestro validation. |
| PromptArtifact → Flow | PARTIAL | Basic facts preserve artifact ID/revision in some paths. | Decomposition ignores `ArtifactID`; complex prompts reduce to heuristic two-step graphs; commands are generic and may not fit the project. |
| Flow/DAG | PARTIAL | Backend/frontend DAG validation and revisions pass tests. | Preflight has hardcoded PASSes; preflight is not mandatory; live edit/run graph and real E2E are unproven. |
| Routing/navigation | MISSING | App reads query flags and local project selection. | No canonical semantic routes, deep links, refresh, back/forward, or project-scoped popout identity. |
| Workspace layout | PARTIAL | Layout model and persistence tests exist. | Backend is not sole presentation authority; presentation remains separate localStorage state; restart and full reset are unproven. |
| Attention/operations | PARTIAL | Radar, notifications, providers and capacity APIs exist. | Activity is not a first-class surface; capacity branch is not integrated; contextual action coverage is incomplete. |
| Settings/accessibility | PARTIAL | Settings, themes and some keyboard behavior exist. | Hardcoded strings/inline styles, incomplete tab semantics, pointer-only resize/drag, and no complete focus verification. |
| Updates/release | PARTIAL | Version/update APIs and cross-build workflow exist. | No verified signed update flow, install smoke matrix, native platform execution, or browser visual/a11y gates. |

## Baseline gates

- `make test-go`: PASS.
- `go test -race ./...`: PASS.
- `go vet ./...`: PASS.
- `make format-check`: PASS.
- `make lint-go`: FAIL with 30 existing lint findings (misspell/staticcheck/unused/unparam/whitespace).
- `make security`: BLOCKED_BY_ENVIRONMENT (`govulncheck` unavailable).
- `npm --prefix web run quality:full`: PARTIAL — checks/tests pass after Bun install, production build fails because `web/src/styles/_tokens.scss` is missing.
- `git diff --check`: PASS at clean baseline.

## Decisions

1. Preserve Mission Runner, DAG validation, Composer persistence, and Agent identity where tests support them; do not rewrite them.
2. Correct P0 safety and build failures before UX expansion.
3. Keep the capacity branch separate until explicitly integrated by the user.
4. Mark native Windows/macOS and browser E2E as unverified unless executed on those platforms.
