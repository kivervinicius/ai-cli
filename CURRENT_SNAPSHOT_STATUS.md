# IAPro Nexus — Current Development Snapshot

Generated: 2026-08-29
Source worktree: feat/nexus-maximum-delivery
Base commit: 829410a60883536064c326501a2de16d0e7a96a8
Original candidate checkpoint: b0cbe285d945165d7e5cb517da63a75a827fe842

This is an **intermediate development snapshot requested by the user**, not the final production-certified release.

## Included progress

- Direct provider/session workflow restored as first-class behavior.
- Agent terminal/runtime continuity hardening.
- Runtime-to-Agent identity mapping corrections.
- Safer Agent configuration/Safe Apply semantics.
- Per-Agent Git worktree isolation groundwork.
- Scheduler quota/capability truthfulness improvements.
- Explicit Intelligence modes and durable clarification groundwork.
- Maestro honesty/degraded-mode corrections in progress.
- Durable Mission Runner/repository/autonomy/scheduling implementation in progress.
- Frontend/API wiring additions and associated tests in progress.

## Base branch

`feat/nexus-maximum-delivery`

## Base commit

`829410a60883536064c326501a2de16d0e7a96a8`

## Working tree at snapshot time

```
 M .gitignore
 M README.en.md
 M README.es.md
 M README.md
 M install.sh
 M internal/app/app.go
 M internal/control/driver/agy_driver.go
 M internal/control/driver/cursor_driver.go
 M internal/control/driver/driver_test.go
 M internal/control/driver/gemini_driver.go
 M internal/control/driver/opencode_driver.go
 M internal/control/launcher/launcher.go
 M internal/control/web/auth.go
 M internal/control/web/dist/bundle.css
 M internal/control/web/dist/bundle.js
 M internal/control/web/e2e_test.go
 M internal/control/web/handlers_nexus.go
 M internal/control/web/server.go
 M internal/core/model/types.go
 M internal/nexus/nexus.go
 M internal/nexus/plan.go
 M internal/nexus/runner/contract.go
 D internal/nexus/runner/lease.go
 M internal/nexus/runner/runner.go
 M internal/nexus/runner/runner_test.go
 M internal/nexus/runner/types.go
 M internal/nexus/runner/verifier.go
 M internal/nexus/store/plans.go
 M internal/nexus/store/plans_test.go
 M internal/tui/tui.go
 M web/src/app/NexusShell.tsx
 M web/src/app/WorkspaceSurfaceHost.tsx
 M web/src/app/modals/MaestroControlModal.tsx
 M web/src/app/modals/WelcomeModal.tsx
 M web/src/features/work/PlanBuilderSurface.tsx
 M web/src/features/work/WorkSurface.tsx
 M web/src/nexus/api.test.ts
 M web/src/nexus/api.ts
 M web/src/types.ts
 M web/src/workspace/WorkspaceTaskbar.tsx
?? internal/control/launchenv/
?? internal/control/originpolicy/
?? internal/core/provider/adapters/cursor/
?? internal/nexus/autonomyguard/
?? internal/nexus/maestrogates/
?? internal/nexus/mission_execution.go
?? internal/nexus/mission_executor.go
?? internal/nexus/mission_repository.go
?? internal/nexus/mission_service.go
?? internal/nexus/runner/executor.go
?? internal/nexus/runner/manual_control.go
?? internal/nexus/runner/repository.go
?? internal/nexus/runner/runner_durable_test.go
?? internal/nexus/runner/verifier_test.go
?? internal/nexus/scheduling.go
?? internal/nexus/store/autonomy.go
?? internal/nexus/store/migrations/0005_autonomy_runtime.sql
?? internal/nexus/store/migrations/0006_mission_schedule_run.sql
?? internal/nexus/store/schedules.go
?? web/src/app/maestroHonesty.test.ts
?? web/src/app/versionHonesty.test.ts
?? web/src/app/workspaceMissionRoute.test.ts
?? web/src/features/work/planBuilderScheduling.test.ts
?? web/src/workspace/taskbarHonesty.test.ts
```

## Important

The repository intentionally contains uncommitted development changes after commit `829410a`. They are part of this snapshot and must not be discarded with `git reset --hard` unless you intentionally want to return to the last checkpoint.

The bundled `nexus` executable may predate some current source changes. Rebuild from source before treating the binary as representative of this snapshot.
