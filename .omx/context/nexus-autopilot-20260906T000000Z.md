# Nexus autopilot context

## Task

Close the IAPro Nexus Composer → canonical Plan → Flow → GO → persistent autonomous convergence path over the existing Nexus and Maestro capabilities.

## Known evidence

- Current branch: `feat/nexus-maximum-delivery`; worktree has substantial pre-existing changes.
- Existing Nexus domains include Composer, Flow/DAG, mission runner, durable repository, scheduler, evidence/verifier, worktree isolation and Maestro availability bridge.
- Maestro source is available at `/projetos/tools/Orquestrador-Maestro`; schemas and planner/runtime tests exist there.
- Existing audit identifies gaps around authoritative readiness, preflight admission, real Maestro plan validation, global convergence, persistence/recovery and browser E2E proof.

## Boundaries

- Preserve all existing user changes; no reset, checkout, clean, commit or push.
- Reuse WorkPlan/FlowRevision and Mission Runner; do not create a parallel orchestration core.
- No provider credentials, production services, or destructive external side effects.
- New frontend styles must be SCSS Modules and UI text must use i18n.

## Current phase

Baseline and gap audit.

## Next action

Run repository and focused Composer/Flow/runner gates, then implement only evidence-backed gaps in small test-first slices.

## Validation

`go test ./...`, `go test -race ./...`, `go vet ./...`, frontend quality/build gates, and focused failure-injection tests where available.
