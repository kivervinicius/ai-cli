# SDD ledger — plan: nexus product finalization

- Base branch: `feat/nexus-maximum-delivery`
- Base SHA: `00539ebdbcbed0478fcbd3c2a8a916410429d55`
- Campaign branch: `feat/nexus-product-finalization`
- Capacity branch `feat/capacity-monitor-notifications` was found but not integrated.
- Audit spec and implementation plan created.

## Wave 1: Router e separação global/projeto (VERIFIED)
- Added `react-router-dom` v7 integration with canonical semantic routing:
  - Global routes: `/`, `/projects`, `/settings`, `/updates`, `/welcome`
  - Project routes: `/p/:projectId/:surface` (`overview`, `work`, `missions`, `terminals`, `agents`, `resources`, `settings`, `maestro`, etc.)
  - Deep linking: `/p/:projectId/work/:sessionId`, `/p/:projectId/missions/:flowRunId`, `/p/:projectId/terminals/:runtimeId`, `/p/:projectId/agents/:agentId`
  - Popout isolation: `/p/:projectId/popout/:surface`
- URL authority: URL controls active project & surface; `localStorage` serves as fallback for root `/`.
- Test suite: `web/src/app/routes.test.ts` (7 tests), `web/src/app/routerIntegration.test.tsx` (3 tests).
- Web full quality gate: `npm --prefix web run quality:full` passed (check:styles, format:check, lint, lint:styles, typecheck, vitest 53 test files / 263 tests passed, build).

## Wave 2: Layout v4, persistência backend, concorrência e restart E2E (VERIFIED)
- SQLite single authority:
  - Migration `0013_layout_v4.sql` adds monotonic `revision` column to `project_layouts`.
  - Store: `SaveLayoutWithRevision` and `GetLayoutRecord` guarantee monotonic revision increment and return `ErrRevisionConflict` when expected revision mismatches.
  - HTTP Handler: `PUT /api/v1/projects/:id/layout` returns `409 Conflict` (`REVISION_CONFLICT`) on concurrent mismatch.
- Frontend Layout v4:
  - `WorkspaceLayoutService`: `WORKSPACE_LAYOUT_VERSION = 4`, with migration path from v1, v2, v3, preserving monotonic revision.
  - `WorkspaceProvider`: tracks active revision and passes expected revision during autosave, handling optimistic concurrency.
  - Popout isolation: popout mode mounts with `saveLayout = undefined` so popouts never overwrite project layout.
- Test suites:
  - Go: `TestProjectLayoutV4MonotonicRevision` in `internal/nexus/store/store_test.go`, layout roundtrip & 409 conflict test in `internal/control/web/handlers_nexus_test.go`.
  - Frontend: `web/src/services/WorkspaceLayoutService.test.ts` (6 tests).
  - All unit & internal tests pass in Go and Vitest.

## Wave 3: Preflight, admission, worktree, autonomy e segurança (VERIFIED)
- Read-only `PlanPreflight` & Fail-Closed `ExecutionAdmission`:
  - `PreflightFlow` executes DAG topological validity, resource availability, git worktree isolation, and autonomy verification checks.
  - `StartMissionRun` and `StartMissionRunApproved` enforce preflight checks and fail closed before executing or creating runner jobs when preflight is not ready.
  - `TestPreflightFlow_RejectsCyclicDAG` and `TestResolveAutonomousExecutionWorkspaceFailsClosedWithoutWorktree` ensure autonomous execution requires dedicated worktree isolation.
- Security & Race tests:
  - `go test -race ./internal/...` passed with 0 data races across all packages.

## Wave 4: Activity/Attention e ações contextuais (VERIFIED)
- SQLite durable events & bus hook:
  - Implemented `EventMetadata`, `RecordEventMetadata`, `ListEventsMetadata` in `internal/nexus/store/events.go`.
  - Added `SetRecorder` hook to `internal/control/events/bus.go` and wired `nexus.Default()` to record all published events durably to SQLite `events_metadata`.
  - Added HTTP endpoint `GET /api/v1/projects/:id/events` in `internal/control/web/handlers_nexus.go` with security redaction and limit filtering.
- Attention radar & contextual actions:
  - Verified frontend `buildAttentionRadar` and `planFocusAttention` in `web/src/app/attentionRadarModel.ts` and `GlobalAttentionRadar.tsx`.
- Test verification:
  - Go: `TestEventsMetadata_RecordAndList` in `internal/nexus/store/events_test.go`, `TestEventBusDurableRecorderHook` in `internal/control/events/events_test.go`, `TestProjectEventsAPI` in `internal/control/web/handlers_nexus_test.go`.
  - Quality gates: `npm --prefix web run quality:full` (53 test files / 264 vitest tests passed, clean build), `go test -race ./internal/...` passed with 0 data races.

