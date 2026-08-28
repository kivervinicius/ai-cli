# AI Control Web Validation Report (Subproject B)

## Executive Verdict: **GO**

The Local Web Control Center (`ai control web`) has been fully designed, implemented, and validated. It operates as a visual client and adapter on top of the exact same underlying AI Control Core, sharing the `RuntimeRegistry`, `RuntimeLauncher`, `ProfileStore`, and `SessionHost` without creating parallel state managers.

---

## 1. Baseline & Branch Stack
- **Parent Branch**: `feat/ai-control-runtime-validation`
- **Current Branch**: `feat/ai-control-web`
- **Commit**: `ea29c86`
- **Go Version**: `go1.25.0 linux/amd64`
- **Frontend Stack**: Bun 1.3.9 + React 19 + TypeScript + xterm.js 5.3 + xterm-addon-fit 0.8

---

## 2. Architecture & Shared Core Evidence
The Web Control Center introduces zero duplicate business logic:
- **Registry**: Dispatches directly to `registry.DefaultRegistry()` (identical to CLI/TUI).
- **Process Supervision**: Spawns processes via `launcher.Default().Launch()` (unified with `ai control start`).
- **Terminal Execution**: Bridges WebSocket directly to `protocol.Client` PTY/ConPTY raw connections.
- **Quota & Diagnostics**: Reuses `quota.Engine` and `driver.DefaultRegistry()`.
- **Self-Contained Executable**: Pre-compiled static distribution (`internal/control/web/dist`) is embedded via `//go:embed all:dist`. No Node.js or external web servers are required at runtime.

---

## 3. Security Hardening & Threat Model
- **Default Loopback Binding**: Binds to `127.0.0.1:<os-assigned-port>`. Never binds to `0.0.0.0` by default.
- **One-Time Cryptographic Bootstrap**: Generates a 256-bit random token on launch (`?token=<hex>`). Upon first visit, the token is consumed, invalidated, and exchanged for an `HttpOnly`, `SameSite=Strict` cookie (`ai_control_session`).
- **CSRF Token Validation**: Every state-changing API request (`POST`, `PUT`, `DELETE`) enforces a matching `X-CSRF-Token` header.
- **Strict Origin Enforcement**: The HTTP handler and WebSocket upgrader reject requests with untrusted or cross-site `Origin` headers (tested and verified with HTTP 403 Forbidden).
- **Workspace Canonicalization**: Process launches are restricted to registered workspaces or canonical paths.

---

## 4. Real Terminal Experience & Single-Writer Lease
- **True Interactive Terminal**: Powered by `xterm.js` with `FitAddon` connecting to `/api/v1/runtimes/:id/terminal`.
- **ANSI & Resize Streams**: Window resize events dynamically propagate rows/cols over WebSocket to PTY/ConPTY.
- **Single-Writer Governance**:
  - The first active connection acquires the writer lease (`CONTROL`).
  - Concurrent observers are designated `VIEW ONLY`.
  - Observers can explicitly request to take control (`Take Control`), which revokes the prior writer lease.
  - Voluntarily yielding (`Release Control`) returns the session to view-only mode.

---

## 5. UI Features & Capabilities
- **Multi-Project Management**: Sidebar lists all detected workspace bindings (e.g. Omega, Omnia, ai-manager) and active runtime counts.
- **Multi-Terminal Tabs**: Independent tabs for each supervised runtime with quick switching.
- **Split-View Terminal Grid**: Supports Single (1x1), Side-by-Side (1x2), and Grid (2x2) layouts.
- **Account Handoff Modal**: Selects alternative profiles for the same provider, previews account details, and executes uninterrupted handoff.
- **Context Continue Modal**: Explicitly indicates *"A NEW SESSION WILL BE CREATED"*, transfers sanitized work checkpoints, and starts target agent thread.
- **Truthful Capabilities Matrix**: Displays live evidence of provider features (Codex, Claude, AGY, Gemini, OpenCode).
- **Audit Event Log**: Real-time timeline of runtime lifecycles and transactions.

---

## 6. Verification & Test Evidence
- **`go test -race ./internal/control/web/...`**: **PASS** (100% pass, 0 data races).
- **`go test -race ./...`**: **PASS** across all packages.
- **`go vet ./...`**: **0 warnings / 0 errors**.
- **Multi-OS Compilation**:
  - `linux/amd64`: PASS
  - `linux/arm64`: PASS
  - `windows/amd64`: PASS
  - `windows/arm64`: PASS
  - `darwin/amd64`: PASS
  - `darwin/arm64`: PASS
- **E2E Test (`TestWeb_FullE2E`)**: Validated server launch, token exchange, session cookie generation, runtime discovery, and WebSocket handshake.

---

## Final Verdict: **GO** (Ready for Subproject C: Private Remote Control)
