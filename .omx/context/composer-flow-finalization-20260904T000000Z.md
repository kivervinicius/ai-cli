# Composer/Flow finalization intake

- Task: finish Composer, Flow, Maestro integration, Agent/runtime UX, tests and evidence.
- Outcome: production-grade, comprehensible prompt-to-workflow experience without breaking Direct mode.
- Provenance: repository branch `feat/autopilot-composer-flow`, HEAD `ebd7264`, clean baseline, source is existing Nexus implementation.
- Constraints: preserve WorkPlan/Mission Runner and Go Supervisor; no destructive git operations; Maestro remains advisory/complementary; no secrets in logs.
- Known facts: Composer persistence and custom command templates already exist; Flow leader policy exists; visual Flow editor, complete materialization, scheduling and full runtime integration remain incomplete.
- Phase: expansion/planning.
- Next action: audit existing contracts and produce scoped implementation spec with explicit partial areas.
- Verify: `go test ./...`, `make web-verify`, `make build`.
