# IAPro Nexus — Finalization Source Baseline

Captured: 2026-08-30 (America/Cuiaba)

## Git

- Branch: `feat/nexus-maximum-delivery`
- Local HEAD: `a1f3a573602a7c43404acf5eb1d4ac0cbc380110`
- Remote feature HEAD after `git fetch origin`: `a1f3a573602a7c43404acf5eb1d4ac0cbc380110`
- Remote `main`: `7ac2776837dc32b809e2d98c3e42fc857d07024a`
- Working tree at capture: clean
- Automatic commit/push: forbidden by the machine constitution

## CI baseline

- Workflow: `.github/workflows/ci.yml`
- Jobs present: `frontend`, `test-linux`, `test-windows`, `test-macos`, `snapshot`
- Snapshot requires all three platform jobs and frontend.
- Same-SHA GitHub run: not queried/available in this local baseline.

## Source-grounded known gaps

- `web/src/workspace/model.ts` declares `WorkspaceModel.version: 1` and has no semantic `logicalKey`/`viewId` split or focused stack.
- `web/src/workspace/state.ts` persists only a `v1` key and rejects any other version.
- `web/src/workspace/WorkspaceProvider.tsx` hydrates and persists in separate effects; persistence is not gated on hydration.
- `internal/app/control_cmd.go` writes JSON doctor output directly to `os.Stdout`.
- Platform-specific Windows runtime behavior cannot be executed on this Linux host; cross-build and focused tests are the available local evidence.

## Existing evidence treated as hypotheses

Existing `DEV/validation/*` reports claim several passes but explicitly mark macOS/Windows runtime, context handoff, Safe Apply, Mission, remediation, restart durability and Take Control as not tested or externally blocked. These claims require fresh evidence before a GO verdict.
