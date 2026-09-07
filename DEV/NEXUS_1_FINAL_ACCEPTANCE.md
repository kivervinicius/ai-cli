# NEXUS 1.0 FINAL ACCEPTANCE

Audit date: 2026-09-07  
Candidate SHA: `5f51d03985f6ca48186f0cb12b3e97a5294600e3` (working tree dirty)  
Branch: `feat/nexus-maximum-delivery`  
Remote: `origin` → `https://github.com/kivervinicius/ai-cli.git`

Fresh remote evidence: CI run [34155789469](https://github.com/kivervinicius/ai-cli/actions/runs/34155789469)
completed for this exact SHA with conclusion `failure` (2026-09-07). The
available GitHub token is invalid, so raw failed logs cannot be downloaded from
this environment; job conclusions remain authoritative and are not treated as
release PASS evidence.

This is an evidence ledger, not a feature inventory. `PASS` requires a fresh
executable result in this checkout. Historical reports are marked as context,
not reused as same-SHA proof.

## Product

| P0 | Status | Evidence |
| --- | --- | --- |
| Resume-first UX | CONDITIONAL | Existing browser/Axe evidence is historical; no fresh same-SHA return-session run |
| Needs You contract | PASS (local) | Structured `runner.HumanIntervention`; runner tests and durable state paths |
| Terminal continuity | CONDITIONAL | Linux continuity tests pass; native Windows/macOS execution unavailable |

## Autonomy

| P0 | Status | Evidence |
| --- | --- | --- |
| Stagnation recovery | PASS (local) | `go test ./internal/nexus/runner`; remediation persists `StrategyVariant` and no-progress tests pass |
| Artifact communication | PASS (local) | Context capsule/work receipt producer-consumer tests; restart persistence tests |
| Worktree safety | PASS (local) | Isolation and canonical-path regression tests; adversarial live two-writer run not executed |
| Agent crash recovery | BLOCKED_EXTERNAL | Requires killing an authenticated provider process and observing recovery lineage |
| Nexus restart recovery | PASS (local) / CONDITIONAL live | Durable runner restart tests pass; active provider restart scenario not run |
| Provider fallback | BLOCKED_EXTERNAL | Requires provider authentication/quota/outage injection |
| Overnight acceptance | PASS (deterministic sandbox) | `TestOvernightAcceptanceSandbox`: parallel plan, dependency receipt, injected failure, remediation and global DoD |
| Real Nexus dogfooding | FAIL | No recorded real self-hosted mission with zero post-GO intervention |

## Platforms

| Surface | Status | Evidence |
| --- | --- | --- |
| Linux | PASS (local) | `go test ./...`, `go vet ./...`, frontend build/typecheck and embedded sync |
| Windows | UNVERIFIED | Cross-build evidence exists; no native ConPTY/Named Pipe/desktop smoke in this environment |
| macOS | UNVERIFIED | Cross-build evidence exists; no native PTY/keychain/desktop smoke in this environment |
| Desktop Linux | CONDITIONAL | Existing Wails evidence is historical relative to candidate SHA |
| Desktop Windows | UNVERIFIED | Native runner unavailable |
| Desktop macOS | UNVERIFIED | Native runner unavailable |

## Quality

| Gate | Status | Fresh evidence |
| --- | --- | --- |
| Frontend typecheck | PASS | `cd web && bun run typecheck` |
| Frontend tests | PASS | `cd web && bun run test` (61 files / 311 tests) |
| Frontend lint/style | PASS | `bun run lint && bun run lint:styles && bun run check:styles` (pre-existing warning only) |
| Frontend build/embed | PASS | `node web/scripts/build.mjs` |
| Backend tests | PASS | `go test ./...` and `make quality` |
| Backend vet | PASS | `go vet ./...` |
| Runner/store race | PASS | `go test -race ./internal/nexus/runner ./internal/nexus/store` |
| Browser E2E/Axe/visual | CONDITIONAL | Historical reports exist; no fresh same-SHA run recorded |
| Security | PASS (local) | `make security` — No vulnerabilities found |
| Packaging | CONDITIONAL | Existing Linux/package evidence; no same-SHA native installer matrix |

## Release

| Gate | Status | Reason |
| --- | --- | --- |
| Same-SHA matrix | FAIL | Exact-SHA CI run 34155789469 failed: Frontend, Windows and macOS failed; Browser/Desktop/Snapshot jobs were skipped |
| Clean install | CONDITIONAL | Linux evidence exists; native install smoke unavailable |
| Upgrade path | CONDITIONAL | Existing tests cover policy; no full three-platform install/upgrade run |
| Release artifacts | CONDITIONAL | Local artifacts exist historically; candidate SHA publication not performed |

## Verdict

`NO_GO`

The local control-plane implementation is buildable and the deterministic safety
paths pass, including a new unattended sandbox with injected failure and global
DoD verification. Release Candidate promotion is still blocked by missing
real-world proof: Nexus dogfooding, authenticated crash/provider recovery,
same-SHA CI, and native Windows/macOS validation remain outstanding. These cannot
be honestly converted to `PASS` from this Linux checkout.

## Exact blockers and reproduction

1. Run the authenticated provider-backed overnight scenario and save mission
   timeline/artifacts; the deterministic sandbox is covered locally, but no
   provider-backed harness is configured here.
2. Run the self-hosted Nexus mission and record mission ID, agents, worktrees,
   artifacts, remediation and global verification on one immutable SHA.
3. Execute the native Windows and macOS jobs (ConPTY/PTY, desktop and installer)
   and attach logs/screenshots to this SHA.
4. Re-authenticate `gh` (current token returns HTTP 403 for logs), inspect run
   34155789469 failure logs, fix the smallest root causes, and rerun all release
   gates on the resulting immutable SHA.
