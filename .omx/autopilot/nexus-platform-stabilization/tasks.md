# Autopilot Tasks

Execution is serial at milestone boundaries. Parallel work is forbidden when ownership overlaps.

| ID | Owner | Scope | Depends on | Verify | State |
|---|---|---|---|---|---|
| T0 | lead | baseline, scope freeze, CI evidence | — | branch/status/SHA/diff check | done |
| T1 | infra | Bun lock, pinned golangci-lint v2.12.2, GoReleaser v2.18.0 | T0 | frozen install, config verify, frontend gates | partial — 59 pre-existing lint findings |
| T2 | runtime | ExecutableResolver for six providers and templates | T1* | resolver table/native tests | done — Linux tests green; native Windows pending |
| T3 | terminal | ConPTY/PTY two-phase lifecycle and cleanup | T2 | bounded echo/resize/close tests | partial — test/close fixes; two-phase API pending |
| T4 | host | startup stages, protocol diagnostics, secure Named Pipe | T3 | injected fault-stage tests | partial — stages/bind ordering; fault taxonomy pending |
| T5 | windows | Job Object process-tree supervision | T4 | child/grandchild/crash tests | partial — hook/kill-on-close compile; native stress pending |
| T6 | paths | PathRef, filesystem identity, SQLite/legacy migration | T1 | alias/collision/security tests | partial — additive schema and Unix identity; Windows native identity pending |
| T7 | security | truthful per-OS credential capabilities | T1 | native capability tests | partial — capability contract and truthful unsupported states |
| T8 | doctor | consolidate existing doctor, JSON, bundle, Web reuse | T2,T4,T6,T7 | read-only/redaction tests | partial — CLI/JSON/bundle/API/Settings UI done; native runtime probes pending |
| G1 | lead | runtime checkpoint | T1-T8 | three complete green CI runs, same SHA | pending |
| T9 | release | nFPM deb/rpm and install smoke | G1 | clean-container install/upgrade/uninstall | pending |
| T10 | update | signed manifest registry and verification | T9 | golden signature/hash/expiry tests | pending |
| T11 | update | apply planner, receipts, rollback | T10 | failure-injection update tests | pending |
| T12 | web | Update Center over existing partial UI/API | T11 | web-verify/component tests | pending |
| T13 | release | tagged workflow, changelog, support evidence | T9-T12 | snapshot/dry-run provenance | pending |
| T14 | lead | final gate and branch protection evidence | T13 | three complete final CI runs | pending |
