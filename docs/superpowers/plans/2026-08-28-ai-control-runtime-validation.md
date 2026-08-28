# AI Control Runtime Validation & Hardening — Execution Plan

## Overview
This plan specifies the task-by-task execution sequence to fulfill the design spec and achieve production-readiness across Milestone Gates M1 through M7.

---

## Milestone 1: Input Safety & Deadlock Elimination (Gate M1)

### Task 1.1: Reproduce & Fix `CmdInput` Deadlock in `SessionHost`
- **Files**:
  - `internal/control/host/host.go`
  - `internal/control/host/host_test.go`
- **Root Cause**: `handleRPCRequest` held `sh.mu.Lock()` when invoking `sh.processAttachedInput()`, which immediately attempted to acquire `sh.mu.Lock()` again. In Go, `sync.Mutex` is non-reentrant, causing an immediate deadlock on `CmdInput`.
- **TDD / Regression Test**: Add `TestSessionHost_CmdInputNoDeadlock` in `host_test.go` with a 2-second timeout context sending `protocol.CmdInput`.
- **Implementation**:
  - Refactor `SessionHost` lock model: split into `stateMu sync.RWMutex`, `clientsMu sync.RWMutex`, and `writerMu sync.Mutex`.
  - Ensure `processAttachedInput` does not deadlock with RPC dispatch.
- **Verification**: `go test -race -v -run TestSessionHost_CmdInputNoDeadlock ./internal/control/host/...`
- **Commit**: `fix(control): resolve re-entrant mutex deadlock in SessionHost CmdInput handling`

### Task 1.2: Implement True Slash Interception Prefix Router
- **Files**:
  - `internal/control/host/slash_prefix.go` [NEW]
  - `internal/control/host/slash_prefix_test.go` [NEW]
  - `internal/control/host/host.go`
  - `internal/control/host/slash_router.go`
- **Root Cause**: Previously, bytes were sent directly to the child process terminal, and on Enter the system attempted to clear the line with `Ctrl+U`. This leaked keystrokes to the child process before Enter.
- **TDD / Regression Test**: Add `TestSlashPrefixRouter_NeverLeaksToChild` sending `"/ai status\r"`, `"//ai prompt\r"`, and `"/help\r"` to verify that `/ai ` never produces writes to the child process.
- **Implementation**:
  - Create `SlashPrefixRouter` with state machine (`IDLE`, `SLASH`, `SLASH_A`, `SLASH_AI`, `CONTROL_BUFFER`, `PASSTHROUGH`, `ESCAPE`).
  - Buffer only while prefix is ambiguous (`/`, `/a`, `/ai`).
  - If input diverges (e.g. `/h`, `/model`), instantly flush buffered prefix bytes and stream all subsequent bytes without buffering.
  - If `//ai` is typed, unescape and send `/ai` to child process.
  - If `/ai ` or `/ai\r` is typed, consume entirely in AI Control and route via `RouteSlashCommand`.
- **Verification**: `go test -race -v -run TestSlashPrefixRouter ./internal/control/host/...`
- **Commit**: `feat(control): implement zero-leak slash prefix router for input interception`

### Task 1.3: Formal Single-Writer Lease & Multi-Observer Architecture
- **Files**:
  - `internal/control/host/writer_lease.go` [NEW]
  - `internal/control/host/host.go`
  - `internal/control/host/host_test.go`
- **TDD / Regression Test**: Add `TestSessionHost_SingleWriterLeaseAndMultiObserver` attaching Writer A, Writer B, and Observer C; verify Writer B cannot inject input until Writer A disconnects or releases.
- **Implementation**:
  - Add explicit lease tracking with connection ID, acquisition timestamps, and release handlers.
- **Verification**: `go test -race -v -run TestSessionHost_SingleWriterLease ./internal/control/host/...`
- **Commit**: `feat(control): implement formal single-writer lease and multi-observer isolation`

---

## Milestone 2: Runtime Supervision & Non-Blocking Fanout (Gate M2)

### Task 2.1: Non-Blocking Per-Client Bounded Fanout
- **Files**:
  - `internal/control/host/fanout.go` [NEW]
  - `internal/control/host/fanout_test.go` [NEW]
  - `internal/control/host/host.go`
- **TDD / Regression Test**: Add `TestSessionHost_SlowObserverDoesNotBlockWriter` where one observer client blocks on read while active writer generates 500KB of output. Verify child process writer never hangs.
- **Implementation**:
  - Create per-client bounded worker queue (`chan []byte`, cap 256).
  - Implement drop policy on queue overflow with client disconnection on prolonged backpressure.
- **Verification**: `go test -race -v -run TestSessionHost_SlowObserver ./internal/control/host/...`
- **Commit**: `feat(control): implement non-blocking per-client bounded fanout queues`

### Task 2.2: Unified `RuntimeLauncher`
- **Files**:
  - `internal/control/runtime/launcher.go` [NEW]
  - `internal/control/runtime/launcher_test.go` [NEW]
  - `internal/app/control_cmd.go`
  - `internal/control/handoff/account.go`
  - `internal/control/handoff/context.go`
- **Implementation**:
  - Unify SessionHost allocation, endpoint discovery, handshake verification, and registry persistence into `RuntimeLauncher`.
  - Re-wire `controlStartCmd`, `PerformAccountHandoff`, and `PerformContextHandoff` to use `RuntimeLauncher`.
- **Verification**: `go test -race -v ./internal/control/... ./internal/app/...`
- **Commit**: `refactor(runtime): unify supervised runtime spawning via RuntimeLauncher`

### Task 2.3: Process Identity & Safe PID Validation
- **Files**:
  - `internal/control/registry/is_alive_unix.go`
  - `internal/control/registry/is_alive_windows.go`
  - `internal/control/registry/cleanup.go`
  - `internal/control/registry/registry_test.go`
- **Implementation**:
  - Validate process start time / creation time and executable identity before attempting kill fallbacks.
- **Verification**: `go test -race -v ./internal/control/registry/...`
- **Commit**: `fix(registry): harden PID recycling verification with start time validation`

---

## Milestone 3: Cross-Platform Windows & IPC Hardening (Gate M3)

### Task 3.1: Windows ConPTY Backend Validation & Honest Fallback
- **Files**:
  - `internal/control/terminal/terminal_windows.go`
  - `internal/control/terminal/terminal_test.go`
  - `internal/control/driver/driver.go`
- **Implementation**:
  - Ensure Windows terminal backend truthfully reports ConPTY capabilities or honest `PARTIAL` when running on non-VT console hosts.
- **Verification**: `go test -v ./internal/control/terminal/...`
- **Commit**: `feat(windows): harden Windows ConPTY terminal backend and capability declarations`

### Task 3.2: Windows Named Pipe E2E Protocol Suite
- **Files**:
  - `internal/control/protocol/protocol_test.go`
- **Implementation**:
  - Add comprehensive multi-client, ping, status, resize, and disconnect tests on Windows Named Pipes and Unix Sockets.
- **Verification**: `go test -race -v ./internal/control/protocol/...`
- **Commit**: `test(protocol): add comprehensive multi-client IPC test suite`

---

## Milestone 4: Account Handoff & Continuity Verification (Gate M4)

### Task 4.1: Mandatory Checkpoint Persistence & Strict Abort
- **Files**:
  - `internal/control/handoff/account.go`
  - `internal/control/handoff/context.go`
  - `internal/control/handoff/checkpoint.go`
  - `internal/control/handoff/handoff_test.go`
- **TDD / Regression Test**: Add `TestAccountHandoff_CheckpointFailureAborts` verifying that disk/permission failures on `SaveCheckpoint` immediately abort transaction at `PREFLIGHT` with `FAILED_SAFE`.
- **Implementation**:
  - Return explicit error from `SaveCheckpoint()` and make checkpoint success a strict prerequisite for state transition.
- **Verification**: `go test -race -v -run TestAccountHandoff_CheckpointFailureAborts ./internal/control/handoff/...`
- **Commit**: `fix(handoff): require successful checkpoint persistence before state transition`

### Task 4.2: Verified Provider Session Continuity & Rollback
- **Files**:
  - `internal/control/handoff/account.go`
  - `internal/control/handoff/discovery.go` [NEW]
  - `internal/control/handoff/handoff_test.go`
- **TDD / Regression Test**: Add `TestAccountHandoff_RollbackOnTargetResumeFailure` and `TestAccountHandoff_VerifySessionContinuity`.
- **Implementation**:
  - Validate that target provider actually resumed the intended session ID before marking handoff as `COMPLETED`.
  - Handle rollback cleanly if target fails.
- **Verification**: `go test -race -v ./internal/control/handoff/...`
- **Commit**: `fix(handoff): implement session continuity verification and transactional rollback`

---

## Milestone 5: Effective Capabilities & Doctor V2 (Gate M5)

### Task 5.1: Dynamic Capability Matrix & Platform Metadata
- **Files**:
  - `internal/control/driver/driver.go`
  - `internal/control/driver/*_driver.go`
  - `internal/app/control_cmd.go`
  - `internal/app/app.go`
- **Implementation**:
  - Remove all platform hardcodes; dynamically report real OS (`runtime.GOOS`), Architecture (`runtime.GOARCH`), and Go version (`runtime.Version()`).
  - Wire `ai control doctor` and `ai version --json` to real build metadata.
- **Verification**: `go test -race -v ./internal/control/driver/... ./internal/app/...`
- **Commit**: `feat(control): derive effective capabilities from live evidence and runtime platform`

---

## Milestone 6: Context Handoff & Secret Redaction (Gate M6)

### Task 6.1: Context Handoff V2 & Redaction Pipeline
- **Files**:
  - `internal/core/security/redact.go`
  - `internal/core/security/security_test.go`
  - `internal/control/handoff/checkpoint.go`
  - `internal/control/handoff/context.go`
- **TDD / Regression Test**: Add test fixtures scanning for OpenAI, Anthropic, Google, AWS, GitHub, JWT, Bearer tokens, and private keys.
- **Implementation**:
  - Enforce bounds on diff stat (max 20KB, max 50 files) and apply universal redaction across all checkpoint metadata.
- **Verification**: `go test -race -v ./internal/core/security/... ./internal/control/handoff/...`
- **Commit**: `test(security): harden secret redaction pipeline and context checkpoint bounds`

---

## Milestone 7: QA, Stress Tests & Final Verification (Gate M7)

### Task 7.1: Full Adversarial QA Suite
- **Files**:
  - `internal/control/host/qa_test.go` [NEW]
- **Scenarios**:
  - Rapid attach/detach spamming.
  - Multi-writer conflict and lease acquisition.
  - Provider unexpected termination and host recovery.
  - Large throughput streaming (10MB) without memory leak.
  - Corrupt registry / state recovery without classic mode regression.
- **Verification**:
  - `go test -race -v ./...`
  - `GOOS=windows go build ./cmd/ai`
  - `GOOS=darwin go build ./cmd/ai`
  - `GOOS=linux go build -o ~/.local/bin/ai ./cmd/ai`
- **Commit**: `test(qa): add comprehensive adversarial QA and concurrency stress test suite`

### Task 7.2: Final Validation Documentation
- **Files**:
  - `AI_CONTROL_FINAL_VALIDATION_REPORT.md` [NEW]
  - `DEV/WORKLOG.md`
- **Commit**: `docs(control): publish final AI Control Runtime Validation report and worklog`

