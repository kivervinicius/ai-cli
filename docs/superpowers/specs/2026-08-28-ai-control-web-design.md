# AI Control Web Control Center — Design Specification (Subproject B)

## 1. Executive Overview
The Local Web Control Center (`ai control web`) is a visual, browser-based management interface that acts as an adapter/client for the existing, unified AI Control Core. It introduces visual project and terminal management without duplicating business logic, storage, or execution planes.

---

## 2. Design Principles
1. **Shared Control Core**: The Web UI uses the exact same `RuntimeRegistry`, `RuntimeLauncher`, `UsageEngine`, `ProfileStore`, and `SessionHost` as the CLI and TUI.
2. **Zero Runtime Dependencies**: The compiled web assets (`web/dist`) are embedded directly into the Go binary (`//go:embed`). Running `ai control web` requires no Node.js, Bun, or external servers.
3. **Local Loopback Security**: The HTTP server binds exclusively to `127.0.0.1` by default on a dynamically assigned port, protected by one-time bootstrap tokens, HttpOnly session cookies, CSRF protection, and strict Origin verification.
4. **Real Terminal Integration**: The Web Terminal uses `xterm.js` over WebSocket connected directly to the `SessionHost` PTY/ConPTY instance, preserving true interactive ANSI rendering, window resizing, and single-writer lease governance.

---

## 3. Architecture & Data Flow

```text
┌────────────────────────────────────────────────────────────┐
│                    Browser Client (React / xterm.js)       │
└─────────────┬───────────────────────────────┬──────────────┘
              │ REST API (JSON)               │ WebSocket
              ▼                               ▼
┌────────────────────────────────────────────────────────────┐
│              Go Web Server (internal/control/web)          │
│  - Embedded Static File Server (//go:embed dist)          │
│  - Security Middleware (Auth, CSRF, Origin, CSP)           │
│  - REST API Handlers (/api/v1/...)                         │
│  - Terminal WebSocket Hub (bridging to SessionHost IPC)    │
└─────────────────────────────┬──────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────┐
│                   AI Control Core Subsystems               │
│  - launcher.RuntimeLauncher                                │
│  - registry.Registry                                       │
│  - quota.Engine & profile.Store                            │
│  - handoff.PerformAccountHandoff / PerformContextHandoff   │
│  - events.DefaultBus                                       │
└────────────────────────────────────────────────────────────┘
```

---

## 4. API & WebSocket Specifications

### 4.1 Security & Authentication
- Bootstrap: Server generates 256-bit random cryptographic hex token upon launch.
- First request: Browser opens `http://127.0.0.1:<port>/?token=<bootstrap_token>`.
- Token Exchange: Server validates token, sets `ai_control_session` (HttpOnly, SameSite=Strict), and invalidates the bootstrap token.
- CSRF: All non-GET requests require the `X-CSRF-Token` header matching the active session.
- Origin Validation: Every request must originate from `http://127.0.0.1:<port>` or `http://localhost:<port>`.

### 4.2 REST Endpoints (`/api/v1/`)
- `GET /api/v1/health`: System health and version.
- `GET /api/v1/workspaces`: Registered workspaces and active project bindings.
- `GET /api/v1/runtimes`: List all supervised runtime sessions.
- `POST /api/v1/runtimes`: Launch a new runtime (`provider`, `profile`, `workspace`).
- `GET /api/v1/runtimes/:id`: Get detailed runtime metadata, health, and effective capabilities.
- `POST /api/v1/runtimes/:id/stop`: Gracefully terminate a runtime.
- `POST /api/v1/runtimes/:id/handoff`: Execute account handoff with session continuity.
- `POST /api/v1/runtimes/:id/continue`: Execute cross-provider context handoff.
- `GET /api/v1/usage`: Aggregated live quota, rate limits, and cooldown status.
- `GET /api/v1/providers`: Truthful provider capabilities and detection status.
- `GET /api/v1/events`: Recent event stream from the Event Bus.

### 4.3 Terminal WebSocket (`/api/v1/runtimes/:id/terminal`)
- Subprotocol frames (JSON or Binary):
  - `output`: Streamed bytes from `SessionHost` stdout.
  - `input`: Keystrokes sent to `SessionHost` (only permitted if client holds writer lease).
  - `resize`: Terminal rows/cols resize request.
  - `lease_acquire`: Client requests interactive writer lease.
  - `lease_release`: Client voluntarily yields writer lease.
  - `lease_status`: Broadcasts whether current connection is `CONTROL` or `VIEW ONLY`.

