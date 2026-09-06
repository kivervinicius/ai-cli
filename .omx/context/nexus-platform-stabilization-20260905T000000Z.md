# Autopilot Context — Nexus Platform Stabilization

- Task: Execute the Nexus Platform Stabilization & Productization plan autonomously.
- Desired outcome: deterministic CI, reliable native runtime lifecycle, truthful diagnostics, signed distribution and safe updates.
- Campaign ID: `01a07224-1c17-7863-881a-6b1963a6ce43`.
- Branch: `feat/nexus-maximum-delivery`.
- Known evidence: Windows ConPTY/Named Pipe/handshake failures; frontend frozen-lockfile drift; unpinned/incompatible Go lint; macOS race evidence missing; doctor and Maestro-only update path exist partially.
- Constraints: do not auto-commit/push; preserve existing dirty changes; no Desktop or new product features; follow local AGENTS and DEV memory; no destructive git operations.
- Touchpoints: `.github/workflows/ci.yml`, `.golangci.yml`, `.goreleaser.yaml`, `internal/runtime`, `internal/control/{terminal,host,protocol,launcher,registry}`, `internal/core/config`, `internal/nexus/store`, `internal/nexus/system_update.go`, `internal/app`, `web/src/features/settings`.
- Current phase: pre-context intake / baseline.
- Next action: capture immutable baseline and create task/checkpoint artifacts before Task 1.
- Validation: `git status --short`, `git branch --show-current`, `git rev-parse HEAD`, `git diff --check`.
