# AI Control Web Control Center — Implementation Plan (Subproject B)

## Overview
This plan implements the Local Web Control Center for `ai-cli` across 5 sequential tasks, connecting the browser interface to the existing Control Core.

---

## Tasks

### Task B1: Web Server Architecture & Security Foundation
- **Files**:
  - `internal/control/web/server.go`
  - `internal/control/web/auth.go`
  - `internal/control/web/server_test.go`
- **Details**:
  - Dynamic loopback listener (`127.0.0.1:0` or explicit `--port`).
  - Bootstrap one-time token generator, HttpOnly cookie session store, CSRF validator, and strict Origin checks.
- **Verification**: `go test -race -v ./internal/control/web/...`

### Task B2: REST API Handlers & Control Core Wiring
- **Files**:
  - `internal/control/web/handlers_api.go`
  - `internal/control/web/handlers_api_test.go`
- **Details**:
  - Expose `/api/v1/workspaces`, `/api/v1/runtimes`, `/api/v1/usage`, `/api/v1/providers`, `/api/v1/events`.
  - Wire to `registry.Registry`, `launcher.RuntimeLauncher`, and `quota.Engine`.
- **Verification**: `go test -race -v -run TestAPI ./internal/control/web/...`

### Task B3: Real Terminal WebSocket Bridge & Lease Governance
- **Files**:
  - `internal/control/web/handler_terminal.go`
  - `internal/control/web/terminal_test.go`
- **Details**:
  - Bidirectional WebSocket proxy bridging `protocol.Client` PTY/ConPTY streams to xterm.js frames.
  - Enforce `CONTROL` vs `VIEW ONLY` lease management.
- **Verification**: `go test -race -v -run TestTerminalWebSocket ./internal/control/web/...`

### Task B4: Frontend Control Center Application (React / xterm.js)
- **Files**:
  - `web/package.json`
  - `web/src/App.tsx`
  - `web/src/components/...`
  - `web/dist/...`
  - `internal/control/web/dist.go` (embedded via `//go:embed`)
- **Details**:
  - Build modern, responsive UI with Project sidebar, Runtime cards, Quota indicators, Split-view xterm.js terminals, and Handoff modals.
  - Compile with Bun into `web/dist`.
- **Verification**: Compile bundle and verify `dist/index.html` embedded in binary.

### Task B5: CLI Integration (`ai control web`) & E2E Validation
- **Files**:
  - `internal/app/control_cmd.go`
  - `internal/control/web/e2e_test.go`
- **Details**:
  - Implement `ai control web [--port <p>] [--no-open] [--listen <addr>]`.
  - Automated E2E verification of full flow.
- **Verification**: `go test -race -v ./internal/control/web/...` and `ai control web --no-open` check.

