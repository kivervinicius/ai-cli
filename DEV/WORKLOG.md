# Worklog: AI CLI Control Plane Evolution

## Date: 2026-08-28
- **Fix: Multi-Quota Awareness & Smart Capacity Selection**:
  - Identified that AGY maintains multiple quota windows (`five_hour`, `weekly`, `claude_five_hour`, `claude_weekly`).
  - Updated `internal/core/provider/adapters/agy/agy.go` to parse all 4 windows (`5h`, `weekly`, `claude_5h`, `claude_weekly`) into `UsageSnapshot.Windows`.
  - Updated `internal/core/quota/quota.go` `GetCachedUsage` to preserve and parse Claude quota windows from legacy `quota.json`.
  - Updated `internal/profile/usage.go` `GetQuotaDetails` to map `ClaudeFiveH` and `ClaudeWeek` windows.
  - Upgraded `internal/core/scheduler/scheduler.go`:
    - Implemented multi-window capacity scoring considering both the critical bottleneck (`minRemaining * 0.6`) and the average availability (`avgRemaining * 0.4`).
    - Scaled capacity score up to `+100.0` points so available tokens heavily drive profile choice.
    - Rebalanced default profile boost to `+5.0` points (strictly used as a tie-breaker on identical capacity rather than overriding accounts with higher availability).
  - Added test cases in `scheduler_test.go` (`TestMultiQuotaBottleneckSelection` and `TestDefaultProfileTieBreaker`).
  - 100% tests passing with `-race` across all packages.
- **Fix: Stdin Lease Injection & Out-of-Band RPC Refactoring**:
  - Replaced raw in-band JSON lease commands in `handler_terminal.go` with out-of-band RPC calls (`protocol.NewClient`).
  - Added defense-in-depth attached mode response suppression for non-interactive commands.
  - Rebuilt offline bundle (`bun run build`) and recompiled `ai` binary.

## Date: 2026-08-27
- **Architectural Refactoring**: Transformed the codebase into a modular local control plane for AI coding CLIs with clean package boundaries (`internal/core/{model,exitcode,security,config,quota,cooldown,classifier,scheduler,fallback,session,telemetry,provider}`).
- **Zero Fake Quotas**: Eliminated all fabricated 100% quota assumptions. Implemented honest usage states (`LIVE`, `CACHED`, `ESTIMATED`, `UNKNOWN`, `UNSUPPORTED`, `RATE_LIMITED`, `ERROR`) with TTL caching and freshness tracking.
- **Smart Account Selection**: Implemented multi-factor scoring selector with hard filters, LRU load-balancing for UNKNOWN states, project bindings, and `ai explain <provider>`.
- **Automatic Fallback & Cooldown**: Implemented cycle-safe retry mechanism with regex-based error classification and rate-limit tracking.
- **5 Providers Supported**: Added full support for Codex, AGY / Antigravity, Claude Code, OpenCode, and Gemini CLI with provider-native session resumption (e.g. `codex resume <id>`).
- **Security & Isolation Presets**: Built isolation policies (`strict`, `developer`, `compat`), secret redaction, and `ai security` audit command.
- **Universal Sessions & TUI**: Implemented cross-provider session index and professional Bubble Tea TUI.
- **Diagnostics & Testing**: Added `ai doctor`, `ai stats`, `ai history`, `ai config validate`, full unit tests passing with race detector and static analysis (`go test -race ./...`, `go vet ./...`).
- **TUI & Quota Enhancements**:
  - Restored quick numeric shortcuts (`1-9`), continue latest session (`c`), interactive quota details modal (`s`), smart resume choice modal (`r`), and official login trigger (`l`).
  - Fixed ANSI slicing visual glitch and aligned columns into a compact layout without box clipping.
  - Implemented legacy `quota.json` fallback in quota engine and provider adapters to display real cached quotas.
  - Fixed AGY authentication detection by scanning `antigravity-oauth-token`, keyring files, and quota files.
- **Documentation Restoration & Evolution**:
  - Restored the comprehensive Portuguese (`README.md`, default) and English (`README.en.md`) documentation files with sanitized examples.
  - Expanded both READMEs with complete guides for the 5 providers, Smart Account Selector (`ai explain`), Honest Quotas (`ai usage`), Universal Sessions, Workspace Bindings (`ai bind`), TUI, Architecture diagrams, and Shell autocompletions.
- **PowerShell Support**:
  - Implemented native `install.ps1` installer for Windows and PowerShell Core (`pwsh`).
  - Added native `ai completion powershell` and `ai completion pwsh` shell completer via `Register-ArgumentCompleter`.
  - Added unit test cases for PowerShell shell completions in `internal/app/app_test.go`.
  - Updated `README.md` and `README.en.md` with PowerShell installation and `$PROFILE` completion guides.
  - Pushed commit `9467f89` to `origin/feat/control-plane-evolution`.
- **1-Line Zero-Clone Installers**:
  - Upgraded `install.sh` to download pre-built release archives with zero dependencies or build from source using Go fallback.
  - Upgraded `install.ps1` for Windows / PowerShell Core (`pwsh`) with release zip downloads, PATH configuration, and fallback.
  - Updated `README.md` and `README.en.md` highlighting one-line installation (`curl -fsSL ... | bash` and `irm ... | iex`).
  - Pushed commit `bde9084` to `origin/feat/control-plane-evolution`.
- **Visual Harmonization & Full Documentation Review**:
  - Restored top banner (`assets/banner.svg`), `style=for-the-badge` shields, language switcher (`🇧🇷 Português | 🇬🇧 English`), and centered styling across both `README.md` and `README.en.md`.
  - Audited all docs in `docs/` (`account-selection.md`, `usage-and-quota.md`, `provider-development.md`, `security.md`, `ai-cli-control-plane.md`, and `ARCHITECTURE.md`), confirming 100% sanitized examples and accurate architecture specs.
  - Pushed commit `85dfd28` to `origin/feat/control-plane-evolution`.
- **Windows Session Discovery & 5-CLI Detailed Documentation**:
  - Implemented multi-root Windows path normalizer (`filepath.ToSlash`), supporting `%USERPROFILE%`, `%LOCALAPPDATA%`, `%APPDATA%`, and `%HOMEDRIVE%%HOMEPATH%`.
  - Added multi-directory session and history resolution in Codex and AGY adapters for Windows native locations (`.codex`, `.gemini/antigravity-cli`).
  - Added dedicated detailed subsections for all 5 supported CLIs (Codex, AGY, Claude Code, OpenCode, Gemini CLI) in `README.md` and `README.en.md`.
  - Pushed commit `008f2f1` to `origin/feat/control-plane-evolution`.

- **AI Control Runtime & Universal Agent Management (Milestones 1-10)**:
  - Baseline audit generated in `DEV/CONTROL_BASELINE_AUDIT.md` from `feat/control-plane-evolution`.
  - Implemented versioned IPC protocol and cross-platform socket/pipe endpoints in `internal/control/protocol`.
  - Implemented persistent runtime registry and lifecycle state engine in `internal/control/registry`.
  - Implemented PTY process coordinator, ring buffer, and universal `/ai` slash command router in `internal/control/host`.
  - Implemented provider control drivers with capability matrix in `internal/control/driver`.
  - Implemented in-memory pub/sub event bus in `internal/control/events`.
  - Implemented safe same-provider account handoff and cross-provider context handoff with secret redaction in `internal/control/handoff`.
  - Created Bubble Tea Control Center TUI in `internal/control/tui`.
  - Integrated `ai control` and `ai ui` CLI commands in `internal/app`.
  - Added deterministic interactive test fixture in `internal/testutil/fakeagent`.
  - Updated `README.md` and `README.en.md` with Section 8 describing Supervised Mode and in-session `/ai` commands.
  - Generated comprehensive implementation report `AI_CONTROL_IMPLEMENTATION_REPORT.md`.
  - All 40+ tests passing with 0 data races (`go test -race ./...`).

- **Fix: PTY Window Size, Raw Mode and Continuous I/O Streaming**:
  - Implemented `pty.StartWithSize` in `internal/control/host/host.go` capturing live terminal rows and cols (with fallback 80x24).
  - Implemented terminal Raw Mode (`term.MakeRaw`) during `attachRuntime` with graceful `defer term.Restore`.
  - Added OS-agnostic window resize signal routing (`protocol.NotifyWinSizeChange` and `client.Resize`) on SIGWINCH.
  - Replaced line-blocking reader with zero-latency continuous byte-by-byte raw streaming in `SessionHost.streamAttachedInput`.
  - Upgraded dependencies cleanly in `go.mod` (`golang.org/x/term`).
  - Tested with `go test -race ./...` (100% pass).

- **Fix: Socket Deadline Timeout & Non-Duplicated Terminal Input Stream**:
  - Removed 5-second deadline expiration in `protocol.Client.Send` by disabling deadlines (`SetDeadline(time.Time{})`) after RPC frames.
  - Added `ClearDeadline()` and `Reader()` methods to `protocol.Client`.
  - Fixed residual buffer flushing in `attachRuntime` to prevent skipping initial PTY output bytes.
  - Fixed slash command input routing in `SessionHost.processAttachedInput` to prevent duplicate line transmission on Enter.
  - Sockets now stay alive continuously for interactive sessions until explicit detach or process termination.
  - Rebuilt binary in `/home/desenvolvedor/.local/bin/ai`.
  - All tests passing with race detector (`go test -race ./...`).

- **Fix: Precision Slash Interception & Readline Buffer Clearing (`Ctrl+U`)**:
  - Implemented `StripANSI` helper in `internal/control/host/slash_router.go` to sanitize escape sequences and control characters before inspecting slash commands.
  - In `internal/control/host/host.go`, when an `/ai` command is detected on Enter:
    - Sends `\x15` (`Ctrl+U`) to the child process PTY to wipe the line from the CLI's readline buffer.
    - Suppresses the `\r` (Enter) from reaching the child process.
    - Broadcasts the formatted AI Control slash response directly to the user terminal.
  - When `//ai` is detected, clears readline with `Ctrl+U` and sends `/ai <text>\r` to child process.
  - Normal commands pass through directly with `\r`.
  - Rebuilt binary in `/home/desenvolvedor/.local/bin/ai`.
  - Tested with `go test -race ./...` (100% pass).

- **Fix: Stale Process Auto-Purge, Smart Stop & TUI Delete Shortcuts**:
  - Implemented `PurgeInactive()` in `internal/control/registry/cleanup.go` to remove ghost/inactive sessions (`STALE`, `FAILED`, `STOPPED`) and delete orphaned socket files.
  - Updated `controlStopCmd` in `internal/app/control_cmd.go` with fallback PID kill and record deletion when socket is unreachable.
  - Added smart stop fallback and new interactive shortcuts `[d/x]` (delete row) and `[c]` (clean all stale) in Bubble Tea TUI (`internal/control/tui/tui.go`).
  - Added unit test `TestPurgeInactive` in `internal/control/registry/registry_test.go`.
  - Rebuilt binary in `/home/desenvolvedor/.local/bin/ai` and purged 7 accumulated ghost records.
  - All tests passing with race detector (`go test -race ./...`).

## Date: 2026-08-28
- **AI Control Runtime Hardening & Truth Audit**:
  - Created new hardening branch `feat/ai-control-runtime-hardening` from published baseline `f54833e`.
  - Conducted full subsystem Truth Audit documented in `DEV/AI_CONTROL_TRUTH_AUDIT.md`.
  - **Windows IPC**: Replaced invalid TCP dial of named pipes with native Windows Named Pipe implementation using `github.com/Microsoft/go-winio` (`D:P(A;;GA;;;OW)`) and Unix Domain Sockets under `$XDG_RUNTIME_DIR` / `/tmp/ai-control-<uid>` with `0600` permissions.
  - **Terminal Backend Abstraction**: Created `internal/control/terminal` defining `Backend` interface with full PTY support on Unix and ConPTY / pipes on Windows.
  - **Truthful Capabilities Framework**: Replaced hardcoded booleans with dynamic `EffectiveCapabilities` containing explicit evidence (`CapabilityStatus`, `Mechanism`, `Reason`, `Tested`). Codex and OpenCode truthfully downgraded to `TERMINAL` mode.
  - **Independent SessionHost Lifecycle**: Implemented hidden background daemon `ai __control-host --runtime <id>` spawned via `Setsid`/`CREATE_NEW_PROCESS_GROUP`, allowing supervised runtimes to persist across client detachments.
  - **Transactional Account Handoff**: Refactored `PerformAccountHandoff` into transactional state machine with mandatory session ID check, target provider validation, preflight checks, pre-stop checkpointing, verified session continuity, and automatic rollback (`FAILED_SAFE`).
  - **Context Handoff V2 & Redaction**: Created `WorkCheckpoint` and `LineageRecord` with bounded file limits and comprehensive secret redaction (`OPENAI_KEY`, `ANTHROPIC_KEY`, `GOOGLE_TOKEN`, `GITHUB_TOKEN`, `AWS_KEY`, `JWT`, `PRIVATE_KEY`). Integrated Smart Account Selector when target profile is omitted.
  - **Universal Slash Quota Integration**: Connected `/ai status`, `/ai accounts`, and `/ai usage` to `quota.Engine` and `profile.GetUsageSnapshot`.
  - **CI Platform Matrix**: Updated `.github/workflows/ci.yml` with `ubuntu-latest`, `windows-latest`, and `macos-latest` matrix testing Go 1.22 and Go 1.24 across feature branches.
  - **Documentation & Reports**: Updated `README.md`, `README.en.md`, marked `AI_CONTROL_IMPLEMENTATION_REPORT.md` as superseded, and generated final hardening report `DEV/AI_CONTROL_HARDENING_REPORT.md`.
  - **Zero Data Races**: All 45+ tests passing with race detector (`go test -race ./...`) and `go vet ./...` clean. Windows cross-compilation verified.


## 2026-08-28: AI Control Full Hardening Execution (All Lanes & Phases)

- **What Changed**:
  - **Lane 1 (Protocol & IPC Security)**:
    - [C-1] Removed silent Named Pipe security fallback in `endpoint_windows.go`.
    - [C-2] Protected Unix socket creation against permission race windows with `syscall.Umask(0177)` and `0600` verification.
    - [C-3] Added ownership verification on `/tmp` socket fallback directory to prevent hijacking.
    - [H-4] Implemented bounded response reading in `protocol.Client` (`readBounded` with 1MB ceiling).
    - [L-1, L-2] Added auto-generated unique request IDs and ticker-based instant context cancellation in `WaitForEndpoint`.
  - **Lane 2 (Registry & Process Lifecycle)**:
    - [H-1] Implemented cross-process file locking for `runtimes.json` using `syscall.Flock` (Unix) and `LockFileEx` (Windows).
    - [H-2] Released registry mutex during network/socket I/O in `CleanupStale` and `PurgeInactive` to eliminate deadlocks.
    - [H-3] Implemented PID recycling validation (`IsProcessAliveWithGeneration`) checking process creation start times against `HostGeneration`.
    - [M-1] Redirected daemon stdout/stderr streams to `<datadir>/logs/<runtime-id>.log`.
    - [H-7] Injected `SIGTERM`/`SIGINT` graceful shutdown traps in `controlHostCmd` and plugged goroutine leaks in `attachRuntime`.
  - **Lane 3 (Handoff, Drivers, Host & TUI)**:
    - [C-4] Reordered `PerformContextHandoff` to verify target runtime is alive before stopping source process (zero session loss).
    - [C-5] Hardened `account.go` rollback to verify source PID liveness before respawning (preventing duplicate processes).
    - [H-5] Added exhaustive integration tests for handoff state transitions, rollback safety, and git bounds.
    - [H-6] Added persistent warning logging for checkpoint and lineage writes.
    - [H-8] Implemented real background handoff execution on intercepted `/ai handoff` and `/ai continue` slash actions.
    - [M-3] Implemented dynamic, capability-aware TUI shortcut rendering (`[h] Handoff` and `[s] Stop`).
    - [M-4] Updated `ARCHITECTURE.md` with complete AI Control Plane section and sequence diagrams.
    - [M-5] Added `BuildKickoffArgs` to `ControlDriver` interface and implemented across all 6 provider drivers.
    - [M-7, L-3, L-4, L-5] Redacted workspace paths, added atomic `.tmp` file writing in checkpoints, cleaned up dead quota init, and removed shadowed `max()` helper.
- **Why**: Production readiness, truthful capabilities, cross-platform security, and zero data-loss resilience across all AI Control operations.
- **Verification**:
  - `go vet ./...` (0 warnings).
  - `go test -race ./...` (100% pass across all packages, 0 data races).
  - Multi-OS build verified (`GOOS=windows`, `GOOS=darwin`, `GOOS=linux`).
  - Binary installed and validated at `/home/desenvolvedor/.local/bin/ai`.

## 2026-08-28: Windows Progress Bar Character Rendering Fix

- **What Changed**:
  - Updated `internal/core/quota/quota.go` (`RenderProgressBar`):
    - On Windows (`runtime.GOOS == "windows"`), replaced UTF-8 block characters `█` (`\u2588`) and `░` (`\u2591`) with standard universal ASCII characters (`#` and `-`, e.g. `[#######---]`), preventing Windows Console / PowerShell / CMD (CP437, CP850, CP1252) from rendering UTF-8 multi-byte sequences as `???` or mojibake.
    - Formatted non-numeric / unknown status states into aligned labels (`[ UNKNOWN  ]`, `[ LIMITED  ]`, `[ UNSUPPORT]`, `[  ERROR   ]`) instead of repeated `?` characters (`[??????????]`), eliminating the appearance of decoding bugs.
  - Updated `internal/tui/tui.go` to use `[ UNKNOWN  ] UNK` fallback instead of `[??????????] UNK`.
  - Updated `internal/core/quota/quota_test.go` and verified with `go test -race ./...`.
## 2026-08-28: Windows Session Resumption Fix (Symlink Privilege Fallback)

- **What Changed**:
  - Implemented `security.SafeLinkOrCopy` in `internal/core/security/link.go`:
    - On Windows, un-elevated non-developer users lack `SeCreateSymbolicLinkPrivilege`. When `os.Symlink` fails, the system automatically falls back to hardlinks (`os.Link`) for files, directory junctions (`mklink /J`) for folders, and recursive copying for cross-volume resources.
  - Updated `internal/core/provider/adapters/codex/codex.go` and `internal/core/provider/adapters/agy/agy.go` to use `SafeLinkOrCopy` and explicitly include the `sessions/` directory in the linked items list.
  - Updated `internal/core/security/isolation.go` to use `SafeLinkOrCopy`.
  - Updated `internal/app/app.go` (`executeResume`) to include clear error context and tips when session resume fails.
- **Why**: Fixes the issue reported on Windows where `ai` failed to resume sessions (`ERROR: No saved session found with ID <id>`) because `CODEX_HOME` did not have access to host sessions due to silent Windows symlink permission failures.
## 2026-08-28: AI Control Runtime Final Validation & Maestro Orchestrated Hardening

- **What Changed**:
  - **Deadlock Elimination**: Fixed re-entrant mutex deadlock in `SessionHost.CmdInput` handling by separating state locks and input handlers.
  - **Zero-Leak Slash Prefix Router**: Implemented `SlashPrefixRouter` state machine in `internal/control/host/slash_prefix.go`, guaranteeing that `/ai <cmd>` never leaks to child provider stdin while streaming normal keystrokes and `//ai` escape without latency.
  - **Bounded Multi-Client Fanout**: Implemented `BoundedFanout` in `internal/control/host/fanout.go` with per-client 256-chunk ring queues and drop policy, preventing slow observers from blocking the terminal writer.
  - **Unified Supervised Launcher**: Created `internal/control/launcher/launcher.go` with `RuntimeLauncher` unifying SessionHost allocation, endpoint discovery, daemon spawning, and handshake verification across `ai control start`, account handoffs, and context continuations.
  - **Mandatory Checkpoint Persistence & Rollback**: Required successful `SaveCheckpoint` before state transitions; verified source process quiescence; enforced safe transactional rollback on target resume failures.
  - **PID Recycling Identity Protection**: Validated start time and host generation in `IsProcessAliveWithGeneration` before executing kill fallbacks in `controlStopCmd` and cleanup routines.
  - **Universal Platform Truth**: Removed hardcoded OS/Arch/Go strings from `ai version` and `ai control doctor`, dynamically reporting `runtime.GOOS`, `runtime.GOARCH`, and `runtime.Version()`. Added provider filtering (`ai control doctor <provider>`).
  - **Adversarial QA Test Suite**: Implemented `internal/control/host/qa_test.go` verifying rapid attach/detach spamming, multi-writer lease handover, and continuous high-throughput streaming.
- **Verification**:
  - `go test -count=1 -race ./...` (44 passed, 0 failed across all packages).
  - `go vet ./...` (0 warnings).
  - Cross-platform compilation: 6 / 6 target platforms (`linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64`) compiled with exit code 0.
  - Documentation: Created `docs/superpowers/specs/2026-08-28-ai-control-runtime-validation-design.md`, `docs/superpowers/plans/2026-08-28-ai-control-runtime-validation.md`, and `AI_CONTROL_FINAL_VALIDATION_REPORT.md`.

## 2026-08-28: AI Control Web Center & Private Remote Control (Subprojects B & C)

- **What Changed**:
  - **Embedded Web Control Center (`ai control web`)**:
    - Created Go HTTP server with dynamic loopback binding (`127.0.0.1:<os-port>`), one-time 256-bit cryptographic bootstrap token exchange, `HttpOnly` `SameSite=Strict` session cookies, and strict CSRF and Origin verification.
    - Integrated React 19 + TypeScript + Lucide + xterm.js SPA compiled with Bun into `web/dist` and embedded directly into the Go binary (`//go:embed all:dist`).
    - Implemented REST API (`/api/v1/workspaces`, `/api/v1/runtimes`, `/api/v1/providers`, `/api/v1/profiles`, `/api/v1/events`) sharing the exact same Control Core without parallel managers.
    - Implemented bidirectional Terminal WebSocket (`/api/v1/runtimes/:id/terminal`) connecting xterm.js to `SessionHost` PTY/ConPTY streams with dynamic window resize forwarding and single-writer lease governance (`CONTROL` vs `VIEW ONLY`).
    - Added UI features: Project sidebar, live runtimes dashboard, multi-terminal tabs, 2x2 split-view grid, Account Handoff modal, and Context Continue modal.
  - **Private Remote Control & Multi-Machine Foundations**:
    - Validated encrypted SSH Port Forwarding Tunnel workflow (`TestRemote_SSHTunnel`) connecting local client browsers to remote host runtimes without exposing public ports.
    - Added private IP range validation (RFC 1918 / CGNAT) and explicit security warning outputs for non-loopback `--listen` bindings.
    - Added `MachineID`, `Location`, and `Transport` fields to `RuntimeSession` with deterministic host identification (`LocalMachineID`).
    - Authored future multi-machine mTLS blueprint in `DEV/AI_CONTROL_REMOTE_NODES_FUTURE.md` and deferred non-goals in `DEV/AI_CONTROL_DEFERRED.md`.
- **Verification**:
  - `go test -count=1 -race ./...` (46 passed, 0 failed across all packages).
  - `go vet ./...` (0 warnings).
  - Cross-platform compilation: 6 / 6 target platforms verified (`linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64`).
  - Reports generated: `AI_CONTROL_WEB_VALIDATION_REPORT.md`, `AI_CONTROL_REMOTE_VALIDATION_REPORT.md`, and `AI_CONTROL_FINAL_ENGINEERING_REPORT.md`.

## 2026-08-28: Multi-Project Persistence, Session Titles, and Conversation Control

- **What Changed**:
  - **Persistent Multi-Project Store (`internal/control/workspace/workspace.go`)**:
    - Created `workspace.Store` persisting registered repositories to `<datadir>/projects.json`.
    - Added REST endpoints `GET /api/v1/workspaces`, `POST /api/v1/workspaces`, `DELETE /api/v1/workspaces?path=...`.
    - Integrated project manager into Sidebar with inline Add Project form, directory validation, and removal.
  - **Session Title Management**:
    - Added `Title` field to `RuntimeSession`, `LaunchOptions`, and `Registry.UpdateTitle`.
    - Added endpoint `POST /api/v1/runtimes/:id/title` for live title editing.
    - Updated `TerminalPane` with click-to-edit inline pencil input and clean header badges.
    - Updated `TerminalView` tabs to display session title, provider, and profile, eliminating the `● ()` empty badge bug.
  - **Target Project Selection on Launch**:
    - Updated `StartModal` to present a project selector choosing among registered workspaces or entering a custom path.
    - Added optional Session Title input when starting new runtimes.
  - **Conversation / Session History & Resume**:
    - Listed both active and past sessions in `Dashboard.tsx`.
    - Added one-click **Resume** button to offline sessions to continue previous conversations seamlessly.
    - Added native driver for **Cursor Agent** (`cursor-agent` / `agent`) with `--resume` / `--continue` support.
    - Enhanced `runtime.LookPath` with proactive multi-path auto-discovery across NVM, Bun, OpenCode, Cargo, and local bin paths.
- **Verification**:
  - `go test -count=1 -race ./...` (47 passed, 0 failed across all packages).
  - Rebuilt and validated Bun bundle in `web/dist`.
  - Rebuilt and installed binary at `/home/desenvolvedor/.local/bin/ai`.

## 2026-08-28: IAPro-Community Identity & Ecosystem Integration

- **What Changed**:
  - **Web Control Center Branding (`web/`)**:
    - Replaced generic icon with vibrant gradient `IAPro` brand emblem in `Sidebar.tsx`.
    - Updated brand typography to `Control Center` with `IAPro Community • v0.4.0` badge.
    - Updated page title in `index.html` to `IAPro Control Center | Agentic Control Plane`.
    - Added direct link to `https://github.com/IAPro-Community` in the sidebar footer and top navigation bar.
  - **Open-Source Documentation (`README.md` & `README.en.md`)**:
    - Added `IAPro-Community` organization badges and official presentation banner.
    - Added dedicated **Ecossistema IAPro Community** section highlighting the integration between **Orquestrador Maestro**, **IAPro Skill Library**, and **IAPro AI Control**.
    - Updated contributing links and guidelines pointing to `https://github.com/IAPro-Community`.
- **Verification**:
  - `go test -race ./...` (47 passed, 0 failed across all packages).
  - Web SPA rebuilt with Bun into `web/dist` and verified embedded in Go binary.
  - Installed updated binary at `/home/desenvolvedor/.local/bin/ai`.

## 2026-08-28: Fix Raw Resize JSON Stdin Injection into Terminal

- **Root Cause Analysis**:
  - When the browser opened a terminal tab or the pane was resized, `TerminalPane.tsx` sent `{ type: "resize", rows, cols }` over WebSocket to `handler_terminal.go`.
  - `handler_terminal.go` called `client.Resize()` using the *already attached* raw streaming connection.
  - `client.Resize()` sent the JSON RPC request (`{"version":1,"command":"resize",...}\n`) down the attached pipe.
  - Because `SessionHost` had switched that connection to raw streaming (`streamAttachedInput`), it treated the incoming JSON bytes as interactive user typing and wrote the entire JSON string into `sh.termBackend.Write` (the agent's PTY stdin).
- **Fix**:
  - **Out-of-Band Control Channel (`handler_terminal.go` & `control_cmd.go`)**: Window resize events now use a separate short-lived control client (`protocol.NewClient(runtimeID).Resize()`), leaving the attached PTY data stream strictly reserved for user keystrokes.
  - **In-Stream Defense-in-Depth Filter (`host.go`)**: `streamAttachedInput` now checks for incoming `protocol.Request` JSON frames; if detected, it handles the RPC internally without ever writing the JSON into the child process PTY. Suppressed echoing RPC response back to attached stdout for `CmdResize`.
- **Verification**:
  - `go test -race ./...` (47 passed, 0 failed).
  - Rebuilt `/home/desenvolvedor/.local/bin/ai`.

## 2026-08-28: Lane 1 - Web Control Center & Secret Sanitization Hardening (P0.1 & P0.2)

- **WebSocket Authentication & Strict Origin Enforcement (P0.1)**:
  - In `internal/control/web/auth.go` and `internal/control/web/handler_terminal.go`:
    - Hardened `CheckOrigin` and `ValidateOrigin` using `net/url.Parse`.
    - Enforced `http` or `https` scheme, extracted `u.Hostname()` and `u.Port()`.
    - Strictly verified `u.Hostname()` is exactly `"127.0.0.1"`, `"localhost"`, or `"::1"`.
    - Explicitly rejected prefix spoofed domains like `http://localhost.evil.com` and `http://127.0.0.1.attacker.com`.
    - Mandated Origin header presence on WebSocket upgrade requests, rejecting empty origins on WebSockets.
    - Configured Gorilla WebSocket upgrader in `handler_terminal.go` to use the exact same strict `CheckOrigin`.
  - In `internal/control/web/server.go`:
    - Protected the WebSocket terminal route `/api/v1/runtimes/:id/terminal` by validating origin and verifying authentication via `s.auth.AuthenticateRequest(r)` before upgrading, returning HTTP 401 Unauthorized for unauthenticated clients.
    - Enforced authentication across all API routes in `s.authMiddleware` (including GET requests: `/api/v1/runtimes`, `/api/v1/workspaces`, `/api/v1/providers`, `/api/v1/profiles`, `/api/v1/events`), while keeping `/api/v1/health` and `/api/v1/session` public.
- **Secret & Environment Sanitization (P0.2)**:
  - In `internal/control/registry/models.go`:
    - Marked `Binary`, `Args`, and `Env` as `json:"-"` in `RuntimeSession`, ensuring raw process environment variables (`os.Environ()`), execution paths, and command arguments are NEVER written to `runtimes.json` on disk or serialized to JSON.
  - In `internal/control/web/handlers_api.go`:
    - Added `sanitizeSession` defense-in-depth sanitization on all runtime endpoints (`GET /api/v1/runtimes`, `POST /api/v1/runtimes`, `GET /api/v1/runtimes/:id`, handoff, and continue).
    - Updated `writeError` to automatically sanitize error strings through `security.Redact`.
    - Redacted audit event summaries in `handleEvents`.
  - In `internal/core/security/redact.go`:
    - Added database URI password redaction (`postgres://user:pass@host:5432/db`).
    - Expanded GitHub token regex coverage (`gho_`, `ghs_`, `ghu_`, `ghr_`).
    - Handled passwords with special characters and added `RedactSlice`.
- **Testing & Verification**:
  - Added unit test cases for unauthenticated GETs (401), public endpoints (200), and prefix-spoofed Origin rejection (403) in `server_test.go`.
  - Added WebSocket unauthenticated (401), spoofed Origin (403), missing Origin (403), and authenticated upgrade tests in `e2e_test.go`.
  - Added persistence sanitization test in `registry_test.go` verifying disk files and JSON marshaling never contain secrets or env vars.
  - Added URI password, diverse GitHub tokens, and `RedactSlice` tests in `security_test.go`.
  - Verified with `go test -count=1 -race ./internal/control/web/...` and `go test -count=1 -race ./...` (All tests PASS with 0 race warnings).




## Date: 2026-08-28 — Final Production Readiness (branch fix/control-production-readiness)
- Fixed P0 runtime crash: `controlStartCmd` nil-deref when provider has no profiles (scheduler returns non-nil result; guards added). Reproduced live, regression test added.
- Runtime IDs: new `internal/control/ids` ULID package (golden vectors, round-trip, uniqueness, sortability); replaced `UnixNano()%100000` and `len(List())+1` everywhere.
- Workspace IDs: canonical-path + SHA-256 (`ws-…`), distinct for same-basename paths; `List()` sorted by LastUsedAt DESC (deterministic).
- Real Windows ConPTY: `terminal_windows.go` rewritten (CreatePseudoConsole/attribute list/CreateProcessW), truthful pipe fallback, `Backend` gained Wait/Signal/Kill/Mechanism; SessionHost lifecycle moved to backend; windows E2E test authored.
- Protocol: bounded frame reader (no unbounded alloc, fuzz 307k), `ERROR_PROTOCOL_VERSION` enforcement (test).
- Handoff: persistent launcher everywhere (Standalone removed from production paths); ResumeVerifier (no blind session-ID copy) before VERIFIED; ULID lineage/checkpoint IDs.
- Security: CSP/XCTO/Referrer/Permissions/frame-ancestors headers; bind policy — public refused, private requires `--remote`, CGNAT 100.64/10; redaction extended (cookies/auth/quoted-JSON keys) + fuzz 149k.
- Version: `internal/buildinfo` single source; fixed const-vs-ldflags bug; VERSION=0.4.0; GoReleaser ldflags verified on release binary.
- Frontend: TS pinned 5.9.3, ESLint + Vitest added (3 tests), lint issues fixed.
- CI: gofmt gate, windows/macos runtime E2E steps, PowerShell smoke, GoReleaser snapshot job.
- Docs: README sandbox/hermetic terminology corrected to credential isolation; gofmt-clean whole repo.
- Reports: DEV/FINAL_BASELINE_AUDIT.md, FINAL_PRODUCTION_AUDIT.md, FINAL_PLATFORM_MATRIX.md, FINAL_PROVIDER_MATRIX.md, FINAL_SECURITY_REPORT.md, FINAL_RELEASE_REPORT.md, FINAL_10_OF_10_SCORECARD.md.
- Scorecard: 87/100 (8.7/10 CONDITIONAL GO). Blockers: Windows/macOS runtime E2E pending CI + local; session expiry/rotation; doctor v2; final independent review.

## Date: 2026-08-28 — Nexus V1 Gate 1 (feat/nexus-v1)
- SQLite durable product state (modernc.org/sqlite, pure Go): internal/nexus/store + embedded idempotent migrations (projects/agents/revisions/generations/lineage/layouts/events/maestro/verification).
- Project domain: ULID id, slug, canonical path (Abs+EvalSymlinks+dir check), repo fields, maestro_mode default ASSIST, MRU ordering.
- Persistent Agent domain: stable agt_ id, project-scoped (IDOR guard), config revisions, runtime generations, lineage, honest continuity statuses.
- Nexus service StartAgent/StopAgent bridging store + RuntimeLauncher.
- Web API: /api/v1/projects* + /api/v1/agents* + agent terminal WS (resolves current generation, 101 verified).
- Frontend: design tokens, UI primitives, AppShell (Nexus+Legacy nav), CommandPalette (Ctrl+K), ProjectsPage (project rail + agents + agent terminal); App.tsx de-goddified.
- Maestro WIP salvage inventory (feat/cli-novo-wip): ~5k LOC planner classified KEEP_IN_MAESTRO; Mission concepts → Gate 7.
- Evidence: go test -race 25 pkgs ok; frontend typecheck/lint/test/build ok; live vertical slice (project→agent→runtime RUNNING→terminal 101→persist across restart).

## Date: 2026-08-29 — Nexus V1 Gates 3-8 + Independent Review

### Gate 3: Agent Configuration
- `AgentConfig` model with 9 config sections (Provider, Profile, Model, Workspace, Isolation, Maestro, Continuity, Environment, Allocation)
- `AnalyzeImpact` returning safe/dangerous/risky change classification with restart requirement
- `SafeApply` with transactional config update, revision creation, and launch compensation on commit failure
- REST API: GET/PATCH `/api/v1/agents/:id/config`, POST `/api/v1/agents/:id/config/apply`, POST `/api/v1/agents/:id/config/impact`
- Frontend: `AgentConfigurationDrawer` with all config sections, impact preview, Apply/Cancel buttons
- TypeScript types: `AgentConfig`, `ConfigImpact`

### Gate 4: Agent Terminal Broker
- `AgentTerminalBroker` (`broker.go`) with per-agent connection management, writer lease governance
- Protocol frames: `CmdRuntimeChanged`, `CmdAgentState`, `CmdContinuityState` with typed payloads
- Nexus observer callbacks: `SetRuntimeObservers` wiring broker to start/stop/recover events
- Frontend: Layout save/restore of open agents in project cockpit

### Gate 5: Resource Scheduler
- `ResourceScheduler` with 4 policies: Balanced, PreserveQuota, PreferProvider, Manual
- Explainable `SchedulerDecision` with score, reason, rejected candidates, and explain path
- Scoring: health bonus, quota awareness, user preference bonus (15% for prefer, 0-1 range)
- Manual policy: rejects all when no explicit preference, score=0 for non-matching
- REST API: GET `/api/v1/resources`, POST `/api/v1/resources/select`
- Frontend: `ResourcePicker` with provider list, health badges, selection feedback

### Gate 6: Maestro Assist
- Maestro contract v1.0.0 with `AdviceRequest`/`AdviceResponse` and `Recommendation` types
- `MaestroClient` with binary discovery, capability query, and degraded fallback
- 3 modes: OFF (unavailable), ASSIST (default), ORCHESTRATE (beta)
- REST API: GET `/api/v1/maestro`, POST `/api/v1/maestro/advice`
- Frontend: `MaestroPage` with status display, degraded fallback, advice request, recommendation cards

### Gate 7: Mission Beta (feature-flagged off by default)
- SQLite migration 0002: `missions`, `mission_tasks`, `mission_assignments` tables
- Mission lifecycle: DRAFT → PLANNING → READY → ACTIVE → PAUSED → COMPLETED/FAILED/CANCELLED
- Task lifecycle: PENDING → READY → ACTIVE → BLOCKED → COMPLETED/FAILED/SKIPPED
- CRUD: CreateMission, GetMission, ListMissions, UpdateMission, DeleteMission
- Tasks: CreateTask, GetTask, ListTasks, UpdateTask with dependencies (JSON array)
- Assignments: CreateAssignment, ListAssignments, UpdateAssignment
- Stats: MissionStats with total/pending/active/completed/failed counts
- REST API: projects/:id/missions, missions/:id, missions/:id/tasks, missions/:id/assign
- Frontend: `MissionsPage` with mission list, detail view, task display, stats

### Gate 8: Web-First Product Completion
- Responsive sidebar: mobile overlay, collapsible with hamburger menu, escape key close
- ARIA labels on navigation, command palette, and interactive elements
- `aria-current="page"` on active nav items
- Missions nav item added to NEXUS_NAV
- Keyboard accessibility improvements

### Independent Review (cavecrew-reviewer)
- 19 findings total: 5 critical, 7 medium, 7 nit
- **Critical fixes applied:**
  1. `prodLauncher.Stop` now returns stop error instead of discarding (nexus.go)
  2. TOCTOU CSRF race in `routeProject` — session captured once (server.go)
  3. TOCTOU CSRF race in `routeAgent` — session captured once (server.go)
  4. DELETE bypass in `handleProjectDetail` — now uses `nexus.DeleteProject` lifecycle guard
  5. DELETE bypass in `handleAgentDetail` — now uses `nexus.DeleteAgent` lifecycle guard
  6. Timeout path now calls `notifyAgentState("FAILED")` (nexus.go)
  7. `stringReader.Read` returns `io.EOF` instead of `fmt.Errorf("EOF")` (maestro.go)

### Verification
- `go build ./...` — clean
- `go test ./... -count=1 -timeout 120s` — 183 passed, 0 failed
- `go vet ./...` — clean
- `bunx tsc --noEmit` — clean
- `bun run build` — 1587 modules, 0.76 MB bundle
- Branch: `feat/nexus-v1`, HEAD: `82470ff6b4b7e368b41917e1d932798d1d327197`

---

## 2026-08-29 — Workspace OS Finalization Autopilot

**Session**: IAPro Nexus Workspace OS Finalization (this session)

**Environment**: Linux/amd64, Go 1.25.0, Node.js 22.17.0, Playwright Chromium headless

### Verification Results

| Check | Result |
|-------|--------|
| ESLint | ✅ 0 errors |
| TypeScript | ✅ 0 errors |
| Vitest | ✅ 9 files / 36 tests |
| Web build | ✅ 590.6kb bundle, dist embedded |
| go vet | ✅ clean |
| go test ./... | ✅ all packages |
| go test -race ./... | ✅ no races |
| Browser QA | ✅ 12/12 viewport×theme |
| Keyboard nav | ✅ verified |
| Demo isolation | ✅ 0 API mutations |
| Security | ✅ no critical gaps |

### Reports Created

- DEV/NEXUS_WORKSPACE_OS_FINAL_QA.md
- DEV/NEXUS_WORKSPACE_OS_VISUAL_QA.md
- DEV/NEXUS_WORKSPACE_OS_ACCESSIBILITY.md
- DEV/NEXUS_WORKSPACE_OS_SECURITY_QA.md
- DEV/NEXUS_WORKSPACE_OS_PLATFORM_MATRIX.md
- DEV/NEXUS_WORKSPACE_OS_FINAL_HANDOFF.md

### Verdict: CONDITIONAL_GO

Linux runtime fully verified. Windows/macOS build-verified only (conditional limitation).
