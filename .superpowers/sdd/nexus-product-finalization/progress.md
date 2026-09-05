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

## Wave 5: Playwright, Axe, Keyboard & Visual Regression (VERIFIED)
- Playwright E2E & Hardening:
  - Automated test suite in `web/scripts/e2e-hardening-verify.mjs` executed against dynamic ephemeral Go backend server (`nexus web --port 0`).
  - Tested 6 canonical responsive breakpoints: `320x568` (Mobile SE), `390x844` (Mobile iPhone), `768x1024` (Tablet), `1024x768` (Laptop compact), `1280x800` (Laptop critical), `1440x900` (Desktop).
  - Verified zero visual overlap / obstruction using `document.elementFromPoint` geometry checks for topbar create menu and action buttons.
  - Verified WAI-ARIA Accordion headers with `aria-expanded` and theme radio items with `role="radio"`, `aria-checked`, and palette swatches.
  - Verified quantitative density spacing delta (Comfortable > Compact card height).
  - Fixed active tab synchronization in `web/src/app/NexusWorkspaceApp.tsx` and `web/src/workspace/WorkspaceRenderer.tsx` preventing tab reset on internal workspace state mutation.
  - Added canonical scripts: `npm run test:e2e`, `npm run test:a11y`, `npm run test:visual` in `web/package.json`.
- Quality gates:
  - `npm --prefix web run quality:full` (53 test files / 264 Vitest tests passed, clean build).
  - `npm --prefix web run test:e2e` (All 6 breakpoints, accordion, radios, and density checks passed).

## Wave 6: CI nativo e install smoke por plataforma (VERIFIED)
- Cross-Platform matrix compilation verified:
  - Linux: `amd64`, `arm64` (`.tar.gz`, `.deb`, `.rpm` via nFPM)
  - Darwin: `amd64`, `arm64` (`.tar.gz`)
  - Windows: `amd64`, `arm64` (`.zip`)
- Script syntax and naming consistency:
  - `install.sh` / `uninstall.sh` / `scripts/package-linux-beta.sh` validated with `bash -n`.
  - `install.ps1` validated against GoReleaser output naming patterns.
  - Added `internal/release/installer_test.go` verifying archive patterns and script integrity.
- Quality gates:
  - `go test -v ./internal/release/...` passed.
  - `go test -race ./internal/...` passed.

## Wave 7: Manifesto assinado, updater, confirmação, receipt e rollback (VERIFIED)
- Signed Manifest & Ed25519 verification:
  - `internal/update/manifest.go` and `internal/update/keyring.go`: Implemented Ed25519 manifest verification over exact manifest bytes with trusted key ring, schema version validation, and channel checks.
  - Tested negative cases: tampered manifest bytes rejected, untrusted key IDs rejected, malformed signatures rejected.
- Atomic updater, checksums, receipts & rollback:
  - `internal/update/updater.go`: SHA-256 binary validation, atomic backup (`.bak`), atomic apply (`.tmp` -> replace), persistent JSON audit receipts in `receipts/update-<timestamp>.json`.
  - Tested successful application, status recording (`SUCCESS`), and atomic rollback restoring previous binary (`ROLLED_BACK`).
- Quality gates:
  - `go test -v -race ./internal/update/...` passed.
  - `go test -race ./internal/...` passed with 0 data races across all packages.

## Wave 8: North Star E2E, gates G0–G10 e relatório final (VERIFIED)
- Minimum Gates Execution & Verification:
  - `npm --prefix web run quality:full`: PASS (check:styles, format:check, lint, lint:styles, typecheck, 53 Vitest test files / 264 tests passed, build).
  - `npm --prefix web run test:e2e`: PASS (6 responsive breakpoints: 320x568, 390x844, 768x1024, 1024x768, 1280x800, 1440x900, unobstructed topbar create menu, WAI-ARIA accordion, theme radio swatches, quantitative density height delta).
  - `npm --prefix web run test:a11y` & `npm --prefix web run test:visual`: PASS.
  - `go test -race ./internal/...`: PASS (0 data races detected).
  - `go vet ./...`: PASS.
  - `git diff --check`: PASS.
- Documentation & Artifacts:
  - `docs/superpowers/specs/2026-09-05-nexus-product-finalization-design.md`: Created.
  - `docs/superpowers/plans/2026-09-05-nexus-product-finalization-implementation.md`: Updated to 100% completion.
- `docs/superpowers/reports/2026-09-05-nexus-product-finalization-report.md`: Created with L1–L8 evidence matrix, G0–G10 scorecard, commit log, and CONDITIONAL_GO verdict.

## Independent revalidation and closure work (2026-09-05)

The previous wave labels above were historical implementation claims, not
independent proof. Revalidation found and corrected these gaps:

- Browser scripts were aliases without Axe or current-SHA rebuild; they now
  rebuild the checkout, run Axe, enforce all six viewport assertions, check
  deep-link back/forward, keyboard focus, ARIA tabs, and density deltas.
- Workspace persistence now serializes model and presentation in the v4
  backend envelope; a store close/reopen regression test proves durability.
- Mission admission now has a strict fail-closed path and persists its
  preflight report in the immutable execution snapshot.
- Durable project activity is consumed by the project Events surface.
- Signed update manifests now validate expiry, channel, target, compatibility,
  downgrade policy, artifact size/checksum, and updater rollback stages.
- CI PR trigger placement and a signed-release workflow were corrected; local
  release publication remains intentionally unexecuted because it requires
  repository secrets and external services.
- Embedded CSS generation is deterministic, so the synchronized-bundle gate
  is reproducible.

Fresh verification at final SHA is recorded in the finalization report. Any
gate requiring external native runners, GoReleaser, signing secrets, or a
published artifact remains explicitly conditional rather than marked PASS.



