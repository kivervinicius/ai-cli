# AI CLI — Production Readiness Design (Final)

- **Date:** 2026-08-28
- **Baseline SHA:** `7ac2776837dc32b809e2d98c3e42fc857d07024a` (`main`)
- **Branch:** `fix/control-production-readiness`
- **Type:** ARCHITECTURAL
- **Status:** APPROVED (user directive "ajuste tudo" after plan review)

## 1. Context

AI CLI is a local control plane for AI coding CLIs (Codex, Claude Code, Gemini CLI,
OpenCode, AGY, Cursor Agent). Three prior subprojects (runtime validation, web control,
private remote) were merged to `main` and "certified GO" — but all runtime evidence was
produced on Linux amd64. Windows and macOS support is declared but not runtime-verified;
several production paths still violate the robustness contract.

This spec converts the user architecture charter into concrete, verifiable requirements.
The binding constraint is **honesty**: no platform, provider, or capability is declared
`SUPPORTED` without fresh runtime evidence, and no claim may exceed what the code does.

## 2. Decisions (locked with maestro)

| Decision | Choice |
|---|---|
| Windows/macOS evidence | CI runtime E2E (authored here) + local confirmation by user |
| Codex / OpenCode control truth | Honest downgrade to `TERMINAL` (structured events only where real) |
| Frontend tooling | Keep Bun + `bun.lock`, pin exact TypeScript, add ESLint + Vitest |

## 3. Architecture invariants

1. **Single RuntimeLauncher** — `ai control start`, Web start, Account Handoff, Context
   Handoff all spawn a persistent `ai __control-host --runtime <id>` daemon. `Standalone`
   is only legal in unit/E2E tests or explicit foreground mode.
2. **Persistent host handshake** — before `RUNNING`: host PID + process identity
   (PID, start time, executable), control endpoint, protocol version, binary version,
   runtime generation token.
3. **Robust IDs** — runtime IDs are ULID (sortable, collision-resistant, URL/filename
   safe). Workspace IDs derive from canonical absolute path + stable hash, never bare
   basename.
4. **Single writer** — `WriterLease` is the only authority for stdin. TUI, CLI attach,
   WebSocket and future remote clients all acquire a lease; second writer is view-only.
5. **Bounded fanout** — each viewer has a bounded queue with defined capacity, drop,
   disconnect and critical-event policy. Slow observers never block provider stdout.
6. **Truthful capabilities** — `EffectiveCapability = ProviderSupports AND
   AIControlImplements AND PlatformSupports AND VersionCompatible AND RuntimeProbePasses`.
   Statuses: SUPPORTED / PARTIAL / UNSUPPORTED / UNKNOWN / NOT_TESTED.
7. **Single version source** — `internal/buildinfo` (Version, Commit, BuildDate, Go
   version, platform), injectable via ldflags; the single truth for `ai version`, health
   endpoint, Web and README.
8. **Loopback-only Web** — default `127.0.0.1`/`::1`; public bind refused; CGNAT
   `100.64.0.0/10` treated as non-public; remote access via SSH tunnel / private VPN.

## 4. Requirements & acceptance

### 4.1 Runtime correctness
- R1 ULID runtime IDs (no `UnixNano()%N`).
- R2 Canonical-path workspace IDs; deterministic MRU ordering by `LastUsedAt DESC`.
- R3 Protocol frame bounds via limited reader; tests for oversized/malformed/partial/
  close/invalid-JSON/unknown-command.
- R4 Protocol version mismatch returns `ERROR_PROTOCOL_VERSION` (no silent continue).
- R5 Writer lease: A=CONTROL, B=VIEW; on A disconnect B upgrades to CONTROL.
- R6 Handoff targets use persistent launcher (no `Standalone`).
- R7 PID reuse protection (start time / creation time).

### 4.2 Platform support
- P1 Linux: PTY, unix socket, SIGWINCH/resize, raw mode, attach/detach/reattach,
  writer lease, browser terminal — E2E in CI (ubuntu).
- P2 Windows: Named Pipe (exists via `winio`), **real ConPTY**
  (`CreatePseudoConsole`/`ResizePseudoConsole`/`ClosePseudoConsole`), resize, attach/
  detach, PowerShell first-class, WebSocket terminal — E2E in CI (windows), no WSL
  requirement.
- P3 macOS: PTY, unix socket, resize, attach, Web, installer, darwin/arm64 — E2E in CI
  (macos) + local user confirmation.
- P4 Centralized platform paths (XDG / macOS conventions / LOCALAPPDATA).
- P5 `install.sh` supports linux amd64/arm64 + darwin amd64/arm64; `install.ps1`
  windows amd64/arm64.

### 4.3 Providers & capabilities
- C1 Detection gated on installed binary + version + platform + probe.
- C2 Truthful matrix; Codex/OpenCode downgraded to TERMINAL (approvals only when a
  programmatic Approve/Reject exists).
- C3 `ai control doctor` reports platform/runtime/IPC/terminal/provider/capability
  status with SUPPORTED/UNSUPPORTED semantics.

### 4.4 Handoff & context
- H1 Account handoff = same provider, different profile/account, same persisted session,
  continuity verified against provider storage (never copied session ID).
- H2 Transactional state machine (REQUESTED→PREFLIGHT→CHECKPOINTED→SOURCE_STOPPED→
  TARGET_STARTING→VERIFYING→VERIFIED→COMPLETED; failure → FAILED_SAFE / ROLLBACK).
- H3 Checkpoint failure and source-not-stopped are hard barriers (ABORT).
- H4 Context handoff = new session; bounded work checkpoint (workspace, branch, status,
  diff stat, changed files, known tests/errors, goal); no invented decisions/tasks/
  reasoning; never transfers chain-of-thought.
- H5 Central redaction pipeline (keys, tokens, Bearer/JWT, .env, .pem, .key, auth.json,
  cookies, credentials) applied to all handoff artifacts and logs; fuzz-tested.
- H6 Lineage persisted: source/target runtime+session IDs, checkpoint ID, relation type,
  timestamp.

### 4.5 Web control
- W1 Web is a client of the same core (no parallel WebRuntimeManager/UsageEngine/Handoff).
- W2 `ai control web` preserved; single binary with embedded frontend.
- W3 Dashboard: Projects, Runtimes, Sessions, Usage, Events, Providers.
- W4 Multi-project grouping by workspace (Omega/Omnia/Infra scenario).
- W5 Multi-terminal: tabs, split right/down, open new window; closing a view = DETACH.
- W6 xterm.js → authenticated WS → SessionHost → PTY/ConPTY → provider.
- W7 Reconnect: runtime survives browser close; bounded replay + live output.
- W8 Actions rendered from EffectiveCapabilities (no fake buttons).

### 4.6 Security & remote
- S1 Loopback default; refuse public bind (error, not warning).
- S2 Bootstrap: cryptographic one-time token → authenticated session → HttpOnly cookie
  (SameSite=Strict; Secure under TLS; scoped Path).
- S3 Session expiration, rotation, logout/invalidation.
- S4 Security headers: CSP, X-Content-Type-Options, Referrer-Policy, Permissions-Policy,
  frame-ancestors.
- S5 CSRF protection on state-changing REST; WebSocket checks session + Origin +
  runtime authorization + writer lease; no wildcard CORS.
- S6 Terminal/provider metadata never rendered as trusted HTML (XSS-safe).
- S7 Remote: SSH tunnel + private VPN documented; CGNAT-aware; no public exposure.

### 4.7 Release / install / version
- V1 Single version source via `internal/buildinfo`; `ai version` JSON.
- V2 Go version contract aligned across go.mod, CI, installers, README (1.25.0).
- V3 GoReleaser snapshot in CI; artifact matrix linux+darwin+windows amd64+arm64;
  checksums.
- V4 `install.sh`/`install.ps1` consistent with artifact naming and Go contract.

### 4.8 QA / observability
- Q1 Fake provider covering: interactive terminal, session ID, resume, wrong resume,
  rate limit, slow/large output, crash, approval.
- Q2 Adversarial: multi-project isolation, multi-terminal (4+), slow observer, rapid
  attach/detach, browser crash, host crash, provider crash, handoff chaos, `-race`,
  fuzz (slash parser, protocol decoder, redactor).
- Q3 Security adversarial: CSRF, bad Origin, unauthenticated WS, session fixation/
  expiration, runtime ID tampering, workspace path traversal, XSS metadata, oversized
  frame, secret leakage.

## 5. Out of scope (YAGNI)

- IDE features (editor, file explorer, debugger).
- Maestro/LLM-router/API-gateway behavior.
- Distributed multi-node control plane (interfaces only: MachineID, RuntimeLocation,
  ControlTransport — already seeded).

## 6. Honesty & scoring

Scorecard sections map to charter §106. Each point requires fresh command evidence.
`UNKNOWN`/`NOT_TESTED` never scores full marks. Final verdict `GO — 10/10` only if
100/100 with 0 P0/P1 and all declared-platform gates pass; otherwise exact lower score
with blockers.

## 7. Definition of done

Spec approved · worktree isolated · TDD proven · P0=0 · P1=0 · Linux/Windows/macOS
evidence per above · handoff + context handoff validated · Web validated · remote
private validated · security validated · release snapshot validated · installers
validated · docs aligned · final reviewer done · scorecard computed · no merge/push/tag.
