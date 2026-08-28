# Final Production Readiness — Execution Ledger

## Baseline Metadata
- **Baseline Branch**: `main`
- **Work Branch**: `fix/control-production-readiness`
- **Baseline Commit**: `7ac2776837dc32b809e2d98c3e42fc857d07024a`
- **Go Version**: `go1.25.0 linux/amd64`
- **OS / Arch**: `linux/amd64` (container)
- **Frontend**: Bun 1.4.0 canonical, TS 5.9.3 pinned, ESLint + Vitest

## Milestone Gates

| Gate | Description | Status | Evidence |
|---|---|---|---|
| F1 | Runtime ID robustness (ULID) | COMPLETE | golden vectors + uniqueness(10k) + sortability tests |
| F2 | Workspace ID canonicalization | COMPLETE | path-hash IDs (distinct same-basename), MRU sort, symlink + full hash |
| F3 | Windows ConPTY (real) | COMPLETE | CreatePseudoConsole + attribute list + CreateProcessW (+ CREATE_UNICODE_ENVIRONMENT); windows E2E test authored; cross-compiles amd64/arm64 |
| F4 | Backend lifecycle consolidation | COMPLETE | Wait/Signal/Kill/Mechanism on Backend; SessionHost driven through backend |
| F5 | Protocol hardening | COMPLETE | bounded frames (fuzz 307k), ERROR_PROTOCOL_VERSION (test) |
| F6 | Handoff correctness | COMPLETE | persistent launcher (no Standalone), source-stop barrier, ResumeVerifier, target-kill on failure, ULID lineage |
| F7 | Security (web + redaction) | COMPLETE | headers live-verified, public bind refused, CGNAT, redaction extended + fuzz 149k |
| F8 | Version/release single source | COMPLETE | internal/buildinfo + ldflags verified on release binary; goreleaser snapshot 6 artifacts + checksums |
| F9 | CI matrix + frontend QA | COMPLETE | windows/macos E2E steps, gofmt gate, snapshot job; eslint+vitest green |
| F10 | Docs truth | COMPLETE | README sandbox terminology corrected; gofmt-clean repo |
| F11 | Independent final review | COMPLETE | 6 P1 + 7 P2 findings → all fixed, re-verified |
| F12 | Scorecard | COMPLETE | 87/100 (8.7/10 CONDITIONAL GO); blockers = Windows/macOS runtime E2E pending CI + local |

## Final evidence

- `go vet ./...` → 0 warnings
- `go test -race ./...` → 24 packages ok, 0 fail
- `gofmt -l .` → empty
- `GOOS=windows go vet ./...` → ok; `GOOS=windows/darwin go build ./...` → ok
- `bun run typecheck/lint/test` → pass (3 unit tests)
- `goreleaser release --snapshot --clean` → 6 artifacts + checksums.txt
- Live: `ai control start fake` → ULID runtime; web headers via curl; bind refusals
