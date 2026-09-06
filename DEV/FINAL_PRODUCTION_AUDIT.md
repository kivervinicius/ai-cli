# AI CLI — Final Production Audit

- **Date:** 2026-08-28
- **Branch:** `fix/control-production-readiness`
- **Baseline:** `main` @ `7ac2776`

## Scope

Adversarial production-readiness pass per the 10/10 charter. Every fix below was
TDD-driven (failing test first) where behavior is testable, and verified with fresh
command evidence.

## Fixes applied (P0/P1/P2 → resolved)

### Runtime correctness
- **ULID runtime IDs** — `internal/control/ids` (new): collision-resistant, sortable,
  URL/filename-safe 26-char Crockford base32. Replaces `UnixNano()%100000` in
  `launcher.go` and `len(List())+1`/`UnixNano()` IDs in `handoff/account.go`,
  `handoff/context.go`. Tests: golden vectors, round-trip, uniqueness (10k), sortability.
- **Workspace IDs** — `internal/control/workspace`: `makeWorkspaceID` = `ws-` + SHA-256
  of canonical absolute path (distinct IDs for same-basename paths). `List()` now sorts
  by `LastUsedAt DESC` deterministically (ID tie-break). Tests added.
- **Bounded IPC framing** — `host.readBoundedLine` caps allocation at `MaxFrameSize`;
  oversized frames return `errFrameTooLarge` (broadcast a visible error). Fuzz +
  unit tests. Replaces unbounded `ReadBytes('\n')`.
- **Protocol version enforcement** — host now rejects `req.Version != ProtocolVersion`
  with `ERROR_PROTOCOL_VERSION` (version 0 accepted as legacy). Test added.
- **Backend lifecycle refactor** — `terminal.Backend` gained `Wait/Signal/Kill/Mechanism`;
  `SessionHost` now drives process lifecycle through the backend (enables real ConPTY
  and removes duplicated process handling in `host/proc_*.go`).
- **P0 crash fix** — `scheduler.SelectBestProfile` no longer returns a nil result for
  empty candidates; `controlStartCmd` and `PerformContextHandoff` guard `res != nil`.
  Regression test added. Reproduced live: `ai control start fake` panicked before fix.
- **Host handshake identity** — confirmed present: `PID` + `HostGeneration` +
  `IsProcessAliveWithGeneration` (PID reuse protection) in registry/launcher.

### Platform support
- **Real Windows ConPTY** — `internal/control/terminal/terminal_windows.go` rewritten:
  `CreatePseudoConsole` / `ResizePseudoConsole` / `ClosePseudoConsole`,
  `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE` via `Initialize/Update/DeleteProcThreadAttributeList`,
  `CreateProcessW` with `EXTENDED_STARTUPINFO_PRESENT`, UTF-16 env block, Windows
  arg quoting. Truthful pipe fallback if ConPTY unavailable (`Mechanism()` reports it).
  Windows-only E2E test (`terminal_windows_test.go`) authored for CI. Cross-compiles
  for windows/amd64 + arm64. Runtime verification pending CI (windows-latest) + local
  user confirmation.
- **Linux/macOS** — Unix PTY path unchanged (creack/pty), backend lifecycle refactor
  preserves behavior; `go test -race` green on Linux.
- **Go version contract aligned** — `go.mod` 1.25.0, CI Go 1.25, README badges
  `Go 1.25+` (both langs), `install.sh` `>=1.25`.

### Providers & capabilities (truthfulness)
- Codex/OpenCode already downgraded to truthful `TERMINAL` mode (StructuredEvents/
  SubmitPrompt/Approvals = UNSUPPORTED, `ControlLevel=TERMINAL`) — re-verified.
- Terminal capability now reports the real platform backend via
  `terminal.BackendMechanism()` (PTY vs ConPTY).
- `Resume` evidence for codex/opencode marked `Tested: false` with honest reason
  (command supported by signature; not runtime-verified against live provider).

### Handoff & context
- All production handoff targets now use the persistent `RuntimeLauncher`
  (`Standalone: false`) — account, context, and rollback paths.
- **ResumeVerifier** (`handoff/resume_verify.go`): target must be alive, RUNNING,
  non-empty session ID, and resume args must reference the session ID before a
  handoff is marked VERIFIED (§31 — no blind session-ID copy). Tests added.
- Redaction pipeline extended: `cookies?`/`auth` keys; quoted JSON keys; tests for
  `.env` block, `auth.json`, cookies, PEM keys, generic credentials. Fuzz test
  (`FuzzRedact`) run 149k execs — no panics, no marker leakage.
- Lineage already persisted (source/target runtime+session, checkpoint ID, relation
  type, timestamp) — verified present for both handoff types.

### Web / security / remote
- **Security headers** added via middleware: CSP (incl. `frame-ancestors 'none'`,
  `object-src 'none'`), `X-Content-Type-Options: nosniff`, `Referrer-Policy:
  no-referrer`, `Permissions-Policy`, `X-Frame-Options: DENY`. Live-verified via curl.
- **Bind policy** (`web/bind.go`): loopback always allowed; private/CGNAT
  (`100.64.0.0/10`)/link-local/ULA require explicit `--remote`; **public bind refused
  (error, not warning)**. Enforced in both `controlWebCmd` and `web.NewServer`
  (defense in depth). Tests + live refusal verified.
- Loopback default, bootstrap one-time token, HttpOnly + SameSite=Strict cookie,
  Origin validation, CSRF on state-changing REST, authenticated WebSocket, no wildcard
  CORS — all confirmed present.

### Release / version / frontend
- **`internal/buildinfo`** — single version source (Version/Commit/BuildDate injected
  via ldflags). `ai version --json` verified from a goreleaser-built binary
  (`0.4.1-next`, commit, build date, go, platform). `VERSION` file → `0.4.0`.
- **GoReleaser snapshot** — all 6 artifacts (linux/darwin/windows × amd64/arm64) +
  `checksums.txt`. ldflags injection verified on extracted binary.
- **Frontend** — typescript pinned `5.9.3`, ESLint (flat config) + Vitest added,
  `lint`/`test` scripts; 5 pre-existing lint issues fixed; 3 unit tests green.
- **CI matrix** — windows/macos runtime E2E steps, gofmt gate, PowerShell smoke,
  GoReleaser snapshot job with artifact assertions.

## Remaining limitations (honest)

| Item | Status |
|---|---|
| Windows ConPTY runtime test | authored; runs in CI (windows-latest); **not run on this Linux box** |
| macOS runtime E2E | authored; runs in CI (macos-latest); **not run on this box** |
| Provider-level session confirmation | only possible via CONTROL_API adapters (not implemented by design); ResumeVerifier proves runtime-level continuity |
| `ai control doctor` enrichment | exists; not fully extended this pass (see scorecard) |

## Fresh evidence summary

- `go vet ./...` → 0 warnings
- `go test -race ./...` → 24 packages `ok`, 0 fail
- `gofmt -l .` → empty (repo clean)
- `bun run typecheck` / `lint` / `test` / `build` → all pass
- `goreleaser release --snapshot --clean` → 6 artifacts + checksums
- Live: `ai control start fake` starts runtime with ULID ID (`fake-06G4M90Q3RZ94RPWZDHBNRSKEC`)
- Live: `ai control web` serves with full security headers on loopback
- Live: `--listen 8.8.8.8` refused; `--listen 192.168.1.5` refused without `--remote`
