# AI Control Truth Audit

## Baseline

- **Branch:** feat/ai-control-runtime-hardening (from feat/ai-control-runtime)
- **SHA:** f54833ef18463e1a60bc4f9f8036025b31ebf51b
- **Remote SHA:** f54833ef18463e1a60bc4f9f8036025b31ebf51b
- **Working Tree:** Clean
- **OS:** Linux (Ubuntu 6.17.0-23-generic x86_64)
- **Go:** go version go1.25.0 linux/amd64
- **Date:** 2026-08-28

## Claim Verification

| Claim | Documentation says | Code actually does | Automated Test | Runtime Test | Verdict |
|---|---|---|---|---|---|
| **Windows Named Pipes IPC** | Named Pipes on Windows (`\\.\pipe\ai-control-<id>`) | `endpoint.go` listens on `127.0.0.1:0` (TCP random port) and dials `\\.\pipe\...` as TCP host, completely broken | `TestEndpointResolution` (only checks path string) | Fails on Windows | FALSE_CLAIM |
| **Windows Terminal / ConPTY** | Terminal & PTY cross-platform supported | Unix uses `creack/pty`; Windows falls back to standard `os/exec` pipes without ConPTY or resize | None on Windows | Fails interactive Windows CLI | FALSE_CLAIM |
| **Terminal Window Resizing** | Dynamic window resizing supported | `NotifyWinSizeChange` is no-op on Windows (`winsize_windows.go`); Unix calls `pty.Setsize` | `TestWinsize` | Valid on Unix only | PARTIAL |
| **SessionHost Persistence on Detach** | Survives client detach, keeps provider running | SessionHost runs inside foreground `ai control start` process; detaching terminates parent or drops control | Unit test in-memory only | Detaching kills host if foreground exits | PARTIAL |
| **Codex `CONTROL_API` & Structured Events** | Codex operates at `CONTROL_API` with structured events, approvals, turn cancellation | `CodexDriver` hardcodes `StructuredEvents: true`, `Approvals: true`, but `BuildCommand` runs plain CLI without `app-server` adapter | `driver_test.go` checks hardcoded struct | Plain CLI execution | FALSE_CLAIM |
| **OpenCode `CONTROL_API` & Server Mode** | OpenCode operates at `CONTROL_API` with structured events | `OpenCodeDriver` hardcodes `StructuredEvents: true`, but `BuildCommand` runs plain CLI without `opencode serve` | `driver_test.go` checks hardcoded struct | Plain CLI execution | FALSE_CLAIM |
| **AGY / Antigravity Support** | Supervised runtime & conversation resume | `AGYDriver` uses `agy --conversation=<id>` and sandbox envs; runs in `TERMINAL` mode | `driver_test.go` | Functional on Unix | SUPPORTED |
| **Claude Code Support** | Supervised runtime & resume | `ClaudeDriver` uses `claude --resume <id>` in `TERMINAL` mode | `driver_test.go` | Functional on Unix | SUPPORTED |
| **Gemini CLI Support** | Supervised runtime & resume | `GeminiDriver` claims `Resume: false`; runs in `TERMINAL` mode | `driver_test.go` | Functional on Unix | SUPPORTED |
| **Account Handoff Integrity** | Verified account transition with preserved session | Hardcoded switch for CLI resume args; allows empty session ID; no transactional rollback; target not verified | `handoff_test.go` (mock registry only) | Not transactionally safe | PARTIAL |
| **Context Handoff & Redaction** | Bounded workspace context with secret redaction | `ExtractContextEnvelope` collects git branch, status; redacts goal/status; missing test observations & diff bounds | `handoff_test.go` | Partial context | PARTIAL |
| **Slash Commands & Quota Integration** | `/ai status`, `/ai usage`, `/ai accounts` show real live quotas | `/ai usage` prints static string; `/ai status` ignores `UsageEngine`; `/ai accounts` lists emails only | `TestSlashRouter` (checks string patterns) | Static/unconnected quota | FALSE_CLAIM |
| **TUI Action Truthfulness** | TUI adapts actions to runtime capabilities | Hardcoded shortcuts `[a]`, `[s]`, `[d]`, `[c]`, `[r]` regardless of provider capability | None | Static actions | PARTIAL |
| **Multi-Writer Protection / Writer Lease** | Single active writer with multiple observers | Multiple connections can send raw bytes simultaneously without lease acquisition | None | Stdin race possible | PARTIAL |
| **CI Platform Matrix** | Cross-platform Linux, macOS, Windows tested | `.github/workflows/ci.yml` only runs `ubuntu-latest` and `macos-latest` on `main`/`master` | Missing Windows CI | Untested on Windows | FALSE_CLAIM |

## Windows IPC
`internal/control/protocol/endpoint.go` implements `Listen` as `net.Listen("tcp", "127.0.0.1:0")` and `Dial` as `net.Dial("tcp", "\\.\\pipe\\ai-control-...")`.
Verdict: FALSE_CLAIM. Windows IPC is completely broken. Requires real Windows Named Pipes implementation with build tags (`endpoint_unix.go` and `endpoint_windows.go` with Microsoft/go-winio or win32 pipe APIs).

## Windows Terminal
Windows execution falls back to standard `os.Pipe` without virtual terminal processing, raw terminal mode, ANSI VT parsing, or resizing support.
Verdict: FALSE_CLAIM. Needs honest ConPTY integration or downgrade of Windows capability declaration to `PROCESS` / `PARTIAL`.

## SessionHost Lifetime
When `ai control start <provider>` is run, `host.NewSessionHost` is executed within the caller's foreground process. Detaching leaves no detached daemon host running in background.
Verdict: PARTIAL. Requires background host lifecycle (`ai __control-host --runtime <id>`) to enable true detachment and reattachment.

## Codex Capabilities
Declared as `CONTROL_API` with `Approvals: true`, `StructuredEvents: true`, and `SubmitPrompt: true`. The code actually executes `codex` directly as a terminal process without interfacing with `codex app-server`.
Verdict: FALSE_CLAIM. Must either implement a real version-guarded `codex app-server` JSON-RPC adapter or truthfully downgrade effective capability to `TERMINAL`.

## OpenCode Capabilities
Declared as `CONTROL_API` with `StructuredEvents: true`. The code actually executes `opencode` directly without `opencode serve` HTTP/SSE server client.
Verdict: FALSE_CLAIM. Must either implement a version-guarded `opencode serve` adapter or truthfully downgrade effective capability to `TERMINAL`.

## Gemini Capabilities
Correctly declared as `TERMINAL` mode.
Verdict: SUPPORTED.

## Claude Capabilities
Correctly declared as `TERMINAL` mode with `Resume: true`.
Verdict: SUPPORTED.

## AGY Capabilities
Correctly declared as `TERMINAL` mode with `Resume: true`.
Verdict: SUPPORTED.

## Account Handoff
1. Allows `source.ProviderSessionID == ""` and continues silently.
2. Input `target` does not validate `targetProvider == sourceProvider`.
3. Resume command generation is hardcoded inside `handoff/account.go` via `switch source.ProviderID` rather than polymorphic driver interfaces.
4. Non-transactional: kills source before verifying target start, no rollback mechanism.
Verdict: PARTIAL. Must be refactored into a transactional state machine with preflight and verified continuity.

## Context Handoff
Extracts git branch, short status, and changed files. Missing diff budget limits, test observations, error observations, and unified redaction pipeline across all envelope fields.
Verdict: PARTIAL. Needs upgrade to `WorkCheckpoint` / `ContextEnvelope` v2.

## Slash Commands
`/ai status`, `/ai usage`, `/ai accounts` are intercepted by `host/slash_router.go` but use placeholder text rather than live engine data.
Verdict: FALSE_CLAIM. Must connect `/ai` router to `quota.Engine` and `profile.GetUsageSnapshot`.

## Quota Integration
Quota statuses `LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`, `UNSUPPORTED` are implemented in `internal/core/quota` but disconnected from `internal/control/host`.
Verdict: PARTIAL.

## TUI
Actions are statically rendered without querying runtime effective capabilities.
Verdict: PARTIAL.

## Security
State directory lacks explicit permission hardening on some paths. Redaction is present in `internal/core/security/redact.go` but must be applied uniformly to all handoff artifacts and logs.
Verdict: PARTIAL.

## CI
Windows is not tested in GitHub Actions. Feature branches do not trigger CI.
Verdict: FALSE_CLAIM.

## Documentation Drift
Reports and READMEs declare `CONTROL_API` for Codex/OpenCode and cross-platform IPC/PTY parity that the code does not satisfy.

## P0 Issues
1. P0-1: Windows IPC Broken (Fix named pipes / build tags).
2. P0-2: Terminal Abstraction & Platform Hardening (PTY / ConPTY / truthful capabilities).
3. P0-3: Truthful Capabilities Framework (EffectiveCapabilities derived from runtime reality).
4. P0-4: Codex / OpenCode Control Truth (Version-guarded adapters or truthful TERMINAL mode).
5. P0-5: SessionHost Independent Lifecycle (Background daemon host `__control-host`).
6. P0-6: Transactional Account Handoff (Preflight, mandatory session ID, continuity verification, rollback).
7. P0-7: Slash Commands Quota Integration (Unified QuotaEngine and UsageSnapshot).
8. P0-8: Windows & Feature Branch CI Matrix.

## P1 Issues
1. P1-1: Context Handoff V2 & Redaction Pipeline (WorkCheckpoint with bounds and redaction).
2. P1-2: Protocol Framing & Stdin Single-Writer Lease (Observer protection and size bounds).
3. P1-3: TUI Capability-Driven Rendering (Dynamic action rendering).
4. P1-4: Doctor Command V2 (Validation status and structured JSON).

## P2 Issues
1. P2-1: Documentation Alignment (Harmonize README, ARCHITECTURE, and new Hardening Report).
2. P2-2: Stale PID & Host Generation Token (Prevent PID reuse issues).
