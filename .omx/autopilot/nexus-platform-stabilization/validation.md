# Validation Log

## T0/T1 — 2026-09-05

- `git branch --show-current` — PASS: `feat/nexus-maximum-delivery`.
- `git rev-parse HEAD` — PASS: `ab88fcbfddb5d99cf77c5e6f651aa2e5aa281770`.
- `git diff --check` — PASS.
- `go test ./internal/core/config ./internal/nexus/store` after path migration — PASS.
- Windows/Darwin cross-compile tests for config/runtime — PASS (compile evidence only).
- `go test ./...` after path and credential changes — PASS.
- `go vet ./...` after path and credential changes — PASS.
- `nexus doctor --json` with isolated temporary directories — PASS.
- `nexus doctor --bundle` — PASS; allowlist contains only `report.json` and `MANIFEST.txt`.
- `go test ./internal/doctor ./internal/app` — PASS.
- `go test ./...` after doctor changes — PASS.
- `go test ./internal/control/web -run TestSystemDoctorAPIUsesSharedReadOnlyReport -count=1` — PASS.
- `bun run typecheck && bun run test -- --run` — PASS: 51 files, 253 tests.
- `GOOS=windows GOARCH=amd64 go build ./cmd/nexus` — PASS (cross-build only).
- `GOOS=darwin GOARCH=amd64 go build ./cmd/nexus` — PASS (cross-build only).
- `go vet ./...` and `git diff --check` — PASS.
- `bun --version` — PASS: `1.3.9`.
- `bun install --frozen-lockfile` — initially failed because lockfile drift existed; after `bun install`, frozen install passes.
- `bun run format:check` — PASS.
- `bun run typecheck` — PASS.
- `bun run lint` — PASS with 36 pre-existing warnings, 0 errors.
- `bun run lint:styles` — PASS.
- `bun run test -- --run` — PASS: 51 files, 253 tests.
- `golangci-lint v2.12.2 config verify` via `go run ...` — PASS.

Pending T1 evidence: pinned Go lint run and frontend build/embed gate. The embedded bundle is already dirty before this campaign, so no build was run yet to avoid overwriting unrelated local work.

- Pinned Go lint run — reproducible failure with 59 existing findings. This is a real quality debt, not a tool-version incompatibility; no findings were suppressed.
- `go test ./...` after T2 — PASS across all Go packages.
- `go test ./internal/control/terminal ./internal/control/protocol ./internal/control/host` — PASS.
- `go test -race ./internal/control/terminal ./internal/control/protocol ./internal/control/host` — PASS.
- `GOOS=windows GOARCH=amd64 go test -c` for terminal, launcher, host and protocol — PASS (compile evidence only; not native runtime evidence).
- Job Object supervision hook — Windows cross-compile PASS; native descendant cleanup evidence pending.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.
