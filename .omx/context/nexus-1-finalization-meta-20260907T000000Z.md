# Context — Nexus 1.0 finalization meta

- Task: traçar e executar uma meta de entrega ponta a ponta com Autopilot e subagentes.
- Desired outcome: Release Candidate comprovado por evidência executável, ou NO_GO com bloqueios reproduzíveis.
- Current evidence: Linux/Web/Go locais verdes; `DEV/NEXUS_1_FINAL_ACCEPTANCE.md` registra bloqueios externos.
- Constraints: preservar MissionRunner, WorkPlan, Flow, Composer, Maestro, provider abstraction e segurança; não auto-commit/push.
- Open blockers: dogfooding real, provider/crash recovery autenticado, same-SHA CI, Windows/macOS nativos.
- Current phase: expansion/planning.
- Next action: executar lanes independentes de evidência e preparar harness same-SHA.
- Validation: `node web/scripts/verify-report.mjs`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `make quality`, `make security`.
