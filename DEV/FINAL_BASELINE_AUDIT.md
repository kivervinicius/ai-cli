# AI CLI — Final Baseline Audit

- **Date:** 2026-08-28
- **Branch:** `fix/control-production-readiness` (worktree `/tmp/opencode/ai-cli-wt`)
- **Baseline:** `main` @ `7ac2776837dc32b809e2d98c3e42fc857d07024a`

## Environment

| Item | Value |
|---|---|
| OS | Linux (container) `6.17.0-23-generic x86_64` |
| Go | `go1.25.0 linux/amd64` |
| Node | `v22.17.0` |
| Frontend tooling | Bun `1.4.0` (canonical), `bun.lock` |
| Git working tree at baseline | CLEAN |

## Baseline validation (pre-change)

| Command | Result |
|---|---|
| `go mod download` | ok |
| `go vet ./...` | 0 warnings |
| `go test -race ./...` | all packages pass (24 packages `ok`) |
| `bun install --frozen-lockfile` (web) | ok |
| `bun run typecheck` (web) | ok (typescript `7.0.2`) |
| `bun run build` (web) | ok (bundle.js 0.73 MB, embedded dist rebuilt) |
| 6-target cross-compile | linux/darwin/windows × amd64/arm64 all build |

## Baseline findings (pre-existing, carried into this phase)

| ID | Area | Finding | Severity |
|---|---|---|---|
| B1 | Version source | `const version = "0.4.0"` in `internal/app/app.go`; ldflags `-X` cannot override a const; `VERSION`=0.1.0; `package.json`=0.4.0 | P1 |
| B2 | Runtime IDs | `UnixNano()%100000` in launcher + handoff; `len(List())+1` in handoff | P1 |
| B3 | Workspace IDs | basename-derived, collisions for same-basename paths; `List()` unsorted | P2 |
| B4 | Windows ConPTY | stubs only (`isConPTY=false` always, plain pipes) | P0 |
| B5 | Web security headers | absent (no CSP/XCTO/Referrer/Permissions/frame-ancestors) | P1 |
| B6 | Bind policy | public bind warned not refused; CGNAT `100.64/10` not handled | P1 |
| B7 | Handoff persistence | `Standalone: true` in account/context handoff targets | P1 |
| B8 | Handoff verification | session ID copied without continuity verification (§31) | P1 |
| B9 | Protocol framing | `ReadBytes('\n')` allowed unbounded allocation before size check | P1 |
| B10 | Protocol version | constant exists but never enforced on the host | P2 |
| B11 | Go version contract | `go.mod` 1.25.0 vs README badges 1.23+ vs `install.sh` >=1.22 | P2 |
| B12 | Frontend QA | no `lint`/`test` scripts; TS `^7.0.2` range pin | P2 |
| B13 | CI | Windows/macOS jobs: vet+test+build only (no runtime E2E); no goreleaser snapshot job; no gofmt gate | P1 |
| B14 | Runtime crash | `controlStartCmd` nil-deref on provider with no profiles | P0 |
| B15 | `gofmt` | repo not gofmt-clean (28 files) | P2 |
| B16 | Docs | README uses "sandbox/hermetic" terminology and overclaims | P2 |

## Verification method

Baseline audit executed in an isolated git worktree on branch
`fix/control-production-readiness`. No commit/push was performed.
