# Tasks — Nexus 1.0 finalization

| ID | Owner | Scope | Depends on | Parallel | Verify / evidence |
|---|---|---|---|---|---|
| E1 | lead | baseline, SHA candidate, ledger, integration | — | no | status, HEAD, ledger |
| E2 | evidence lane | audit P0 blockers and native prerequisites | E1 | yes | blocker matrix + reproduction commands |
| E3 | test lane | deterministic overnight, crash/restart/provider fixtures | E1 | yes | focused Go acceptance tests |
| E4 | runtime lane | artifact/worktree/stagnation/NeedsHuman contract audit | E1 | yes | runner/store tests + race |
| E5 | UX lane | Resume-first/Needs You/browser accessibility evidence | E1 | yes | Playwright/Axe/visual on candidate SHA |
| I1 | lead | integrate disjoint patches and reconcile conflicts | E2-E5 | no | diff review + focused regression |
| Q1 | lead | quality cycle, max 5 iterations | I1 | no | web verify, Go quality/security |
| V1 | reviewers | functional, security, code review | Q1 | yes | three approvals or findings |
| V2 | release lane | same-SHA CI, native Win/mac, packaging/install | V1 | yes | immutable SHA matrix |
| R1 | lead | dogfood real Nexus mission and finalize ledger | V2 | no | mission timeline, artifacts, verification, verdict |
| C1 | CI owner | land formatting fix, browser E2E fix and Bash-3.2 macOS workflow fix; inspect macOS race/Windows test failures | auth + E2/E5 | no | new SHA: Frontend/browser/Desktop macOS PASS, native test failures triaged, all required jobs successful |

## Gate policy

Cada tarefa deve entregar paths alterados, comandos, resultado, blockers e risco residual. Falha repetida três vezes vira blocker explícito; nunca vira PASS por relaxamento.

Antes de `I1`, o lead deve revisar sobreposição de paths e exigir SHA, timestamp
e artefato anexado para cada evidência. Blockers externos seguem owner e
ambiente definidos em `DEV/NEXUS_1_DELIVERY_META.md`; ausência de ambiente é
`BLOCKED_EXTERNAL`, nunca aprovação implícita.
