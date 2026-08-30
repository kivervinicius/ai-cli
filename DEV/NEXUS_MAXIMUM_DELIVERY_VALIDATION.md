# IAPro Nexus — Maximum Delivery Validation

**Date:** 2026-08-29
**Branch:** `feat/nexus-maximum-delivery`
**Candidate version:** `0.5.0-beta.5`
**Implementation plan:** `docs/superpowers/plans/2026-08-29-nexus-maximum-delivery.md`

## Verdict

**CONDITIONAL_GO** for a controlled local beta / validation candidate.
**Not yet certified as a production v1.0 release.**

The condition is evidence, not missing product intent: this sandbox has Go 1.23.2, cannot download Go 1.25 or uncached Go dependencies, has no authenticated coding provider, and cannot physically execute macOS/Windows runtime tests.

## Fresh evidence in this environment

### Passed

- `git diff --check`
- `gofmt -l internal cmd` -> empty
- `go test ./internal/nexus/runner -count=1` with temporary Go-1.23-compatible modfile -> PASS
- `go test ./internal/nexus/autonomyguard -count=1` with local dependency stubs -> PASS
- `go test ./internal/control/driver -count=1` with local dependency stubs -> PASS
- `go test ./internal/profile -count=1` with local dependency stubs -> PASS
- targeted Web tests for session rotation/logout, entropy failure and project routing -> PASS
- origin policy tests -> PASS
- protocol + terminal Linux tests available with local stubs -> PASS
- frontend TypeScript -> PASS
- frontend ESLint -> PASS
- frontend Vitest -> **22 files / 76 tests PASS**
- frontend build -> PASS
- `web/dist` and `internal/control/web/dist` are byte-identical by SHA-256 after build
- local real-provider smoke harness `scripts/nexus-e2e-local.go` compiles against an API-compatible WebSocket stub

### Explicitly not certified here

- `go vet ./...` using Go 1.25
- `go test ./...` using Go 1.25 + real `modernc.org/sqlite`, `gorilla/websocket`, PTY and platform dependencies
- `go test -race ./...` using Go 1.25
- physical Windows ConPTY + Named Pipe runtime
- physical macOS PTY + Unix socket runtime
- real provider Direct Work smoke, because this sandbox has no authenticated provider/profile
- full GitHub Actions matrix execution from the final commit

## Product-critical semantics verified in code/tests

1. Direct Work remains independent of Mission, Intelligence and Maestro.
2. `AgentID` is stable; `RuntimeID`/RuntimeGeneration are ephemeral and explicitly linked.
3. Runtime restart/reconfiguration does not claim `LIVE_SAME_RUNTIME` for a new process.
4. Mission packages use staged execution, bounded retries, verification and durable evidence.
5. `COMPLETED_VERIFIED` requires verification enabled and available.
6. WorkPlan dependencies are normalized and validated as a DAG before persistence.
7. WorkPlan updates use optimistic concurrency and reject stale revision overwrites.
8. Mission lease acquisition uses fencing and rejects live reacquisition, including same-owner concurrency.
9. Take Control requires a real allocated Agent and records a durable manual checkpoint.
10. Scheduling claims a due schedule atomically before run creation/execution.
11. Session rotation generates replacement entropy before revoking the current browser session.
12. Project Mission routes are actually dispatched by the server router.
13. Cursor profile/account discovery is connected to resource discovery/Scheduler.
14. The stale tracked Linux `nexus` binary was removed; source ZIPs cannot accidentally ship an executable built from an older commit.

## Promotion gate

This candidate can only be promoted from `CONDITIONAL_GO` to `GO` after the local validation prompt passes on Go 1.25 and the CI matrix proves Linux, macOS and Windows on the final clean commit.
