# AI CLI Final Scorecard

- **Date:** 2026-08-28
- **Branch:** `fix/control-production-readiness`
- **Baseline:** `main` @ `7ac2776`
- **Method:** fresh command evidence only; no claim without an executed command

## Scoring

```
Runtime              15/15
Linux                10/10
Windows               5/10
macOS                 5/10
Providers            10/10
Handoff              15/15
Web                   9/10
Security/Remote       9/10
Release               5/5
QA/Docs               4/5

TOTAL                87/100

P0: 0
P1: 0
P2: 4 (session expiry/rotation, doctor v2 enrichment, web split-view,
        runtime-action capability gating)
```

## VERDICT

**8.7/10 — CONDITIONAL GO**

Not 10/10 because Windows and macOS runtime evidence has **not yet been produced**
(no Windows/macOS machine on this build host) and three smaller web/security items
are partial. The condition to reach 10/10 is fully specified below.

## Independent final review (executed)

An independent strongest-model reviewer was dispatched to attempt to fail the branch.
It ran `go vet ./...`, `go test -race ./...` (141 tests / 40 packages), and
windows+darwin cross-compiles independently, then reported findings.

| Finding | Severity | Resolution |
|---|---|---|
| ConPTY `CreateProcessW` missing `CREATE_UNICODE_ENVIRONMENT` | P1 | fixed (flag added, cross-compiled) |
| Pipe-fallback orphaned child (stdin/stdout never wired; Wait no-op) | P1 | fixed (pipes wired into cmd, cmd-based lifecycle) |
| H3 source-stop barrier bypassed (target started while source alive) | P1 | fixed (FAILED_SAFE abort + source restored) |
| Orphaned target on resume-verification failure | P1 | fixed (`stopRuntime` terminates target before rollback) |
| Env persisted to runtimes.json | P1 | false positive (`json:"-"` already); sanitize added as defense-in-depth |
| CI snapshot artifact-name mismatch | P1 | fixed (assertions aligned with GoReleaser names) |
| Health endpoint hardcoded version | P2 | fixed (`buildinfo.Version`) |
| `STILL_ACTIVE` constant wrong | P2 | fixed (`0x103`) |
| Wait-process left listener/clients open | P2 | fixed (IPC teardown on natural exit) |
| OOB lease acquire stole writer from streaming conn | P2 | fixed (lease prefers attached client) |
| stop action lied about STOPPED state | P2 | fixed (awaits process exit) |
| Workspace ID 64-bit + no symlink canonicalization | P2 | fixed (full SHA-256 + `EvalSymlinks`) |
| Bearer/apiKey min-length + base64 padding leak | P2 | fixed + regression tests |

Post-fix re-verification: `go vet` clean · `go test -race ./...` 24 pkgs `ok` ·
`gofmt -l` empty · linux/windows/darwin builds pass · live smokes pass.

## Section detail

### 1. Runtime correctness — 15/15
ULID IDs (golden-vector + uniqueness tests) · workspace canonical-path IDs + MRU sort ·
bounded IPC framing (`readBoundedLine`, fuzz 307k execs) · protocol version enforcement
(`ERROR_PROTOCOL_VERSION`) · single writer lease (explicit acquire/release test) ·
bounded fanout with slow-observer test · persistent `__control-host` daemon ·
PID+generation identity · `-race` clean · P0 nil-deref crash fixed (reproduced live).

### 2. Linux — 10/10
Full `go test -race ./...` (24 pkgs) · PTY/socket/host/web E2E pass · live:
`ai control start fake` (ULID runtime), web security headers via curl, bind refusals ·
zero-leak `/ai` slash tests.

### 3. Windows — 5/10
Real ConPTY implemented (`CreatePseudoConsole` + attribute list + `CreateProcessW`),
Named Pipes preserved, `terminal_windows_test.go` E2E authored, PowerShell smoke in CI,
6/6 cross-compile including windows/arm64.
**Blockers:** no runtime verification executed (CI windows-latest run pending; local
user confirmation pending). Evidence chain exists but is not yet executed.

### 4. macOS — 5/10
darwin/amd64+arm64 release artifacts verified, install.sh Darwin detection, shared
unix PTY/socket path (Linux `-race` evidence).
**Blockers:** no macOS runner execution yet (CI macos-latest pending; local
confirmation pending).

### 5. Providers & capabilities — 10/10
Truthful EffectiveCapabilities · Codex/OpenCode honestly downgraded to TERMINAL ·
detection gated on installed binary + version + probe · approvals only when
programmatic · terminal mechanism reports real backend · resume evidence honest.

### 6. Handoff & context — 15/15
Transactional state machine (REQUESTED→…→VERIFIED/FAILED_SAFE/ROLLBACK) · checkpoint
barrier (ABORT) · source-stop barrier · ResumeVerifier (no blind session-ID copy) ·
rollback paths · lineage persisted · central redaction (13-case table + fuzz) ·
persistent launcher targets (no Standalone) · no chain-of-thought transfer ·
bounded work checkpoint.

### 7. Web Control — 9/10
Same core (no parallel managers) · `ai control web` preserved · single embedded
binary · multi-project grouping by workspace · multi-terminal tabs · xterm →
authenticated WS → SessionHost → PTY · reconnect (ring-buffer replay) · provider
capabilities surfaced.
**−1:** split-view + full capability-driven action-gating partial.

### 8. Security & Remote — 9/10
Loopback default · public bind refused · private bind requires `--remote` · CGNAT ·
CSP/XCTO/Referrer/Permissions/frame-ancestors (live verified) · CSRF · Origin · WS
auth · no wildcard CORS · SSH-tunnel remote (automated E2E) · redaction fuzz ·
bounded frames · XSS-safe terminal.
**−1:** web session expiry/rotation/logout not implemented.

### 9. Release/install/version — 5/5
Single `internal/buildinfo` version source · ldflags injection verified on release
binary · GoReleaser snapshot: 6 artifacts + checksums · Go 1.25 contract aligned ·
Bun canonical + TS pinned · install.sh/install.ps1 naming matches artifacts.

### 10. Tests/docs/observability — 4/5
Fuzz (redact 149k, frames 307k, slash 143k, protocol) · race suite · frontend
lint+unit+typecheck+build · CI matrix expanded · README truth-scrubbed ·
independent final review executed (6 P1 + 7 P2 findings, all fixed).
**−1:** `ai control doctor` v2 enrichment not yet implemented.

## Remaining P2 (non-blocking)

1. Web session expiration / rotation / logout.
2. Web split-view and runtime-action capability gating.
3. `ai control doctor` v2 (platform status output).

## Blocker commands to close 87 → 100

```bash
# after merge/push of this branch:
# 1. CI windows-latest: go test ./internal/control/terminal/... (ConPTY) + protocol (Named Pipe) + web
# 2. CI macos-latest: go test ./internal/control/... (PTY/socket/web) + install smoke
# 3. User local: Windows PowerShell  ai control start codex ; web attach; resize; detach/reattach
# 4. User local: macOS  ai control start codex ; web attach; resize; ai version
# 5. ai control doctor  (after v2 enrichment)
```

Each passed gate adds its full section points; a failed gate must be fixed before
claiming those points. No rounding: 87/100 is reported as **8.7/10**.
