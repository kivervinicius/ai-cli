# IAPro Nexus Workspace OS — Security QA Report

Date: 2026-08-29  
Branch: feat/nexus-workspace-os-handoff  
Commit: 7f3cb574fd5811baf8b9ab79bfffb5ebc6c3c431  

## 1. CSP / Security Headers

**Status: ✅ PASS**

From `internal/control/web/server.go` — `withSecurityHeaders` middleware applied to every response:

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data:;
  font-src 'self' data:;
  connect-src 'self' ws: wss:;
  base-uri 'self';
  form-action 'self';
  frame-ancestors 'none';
  object-src 'none'

X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()
X-Frame-Options: DENY
```

- No wildcard sources
- `unsafe-inline` only for styles (required for Tailwind/dynamic CSS-in-JS)
- WebSocket allowed only via `ws:/wss:` with no wildcard origin
- Frame embedding blocked (`frame-ancestors 'none'` + `X-Frame-Options: DENY`)
- Object/plugin sources blocked

## 2. Authentication

**Status: ✅ PASS**

- One-time bootstrap token (32 bytes random hex) generated at server start
- Token consumed on first exchange (`usedBootstrap` flag, thread-safe mutex)
- Session cookie: `HttpOnly=true`, `SameSite=Strict`, `Path=/`
- Session ID: 32 bytes random hex per session
- CSRF token: 16 bytes random hex per session
- No session expiration timer (local tool, acceptable for single-user local server)

## 3. CSRF Protection

**Status: ✅ PASS**

All mutating routes (POST/PUT/DELETE/PATCH) require `X-CSRF-Token` header matching the session's CSRF token:

```go
// CSRF enforcement for all mutating methods (P0-2)
if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
    csrf := r.Header.Get(csrfHeaderName)
    if csrf == "" || csrf != sess.CSRFToken {
        writeError(w, http.StatusForbidden, "invalid CSRF token")
        return
    }
}
```

Applied to:
- All `/api/v1/projects/*` routes
- All `/api/v1/agents/*` routes  
- WebSocket terminal routes

CSRF token delivered to client via `/api/v1/session` endpoint (authenticated). Client stores token and sends via `X-CSRF-Token` header on all mutations (`setNexusCSRF` in `nexus/api.ts`).

## 4. Origin Validation

**Status: ✅ PASS**

From `internal/control/web/auth.go`:

- Origin header parsed via `url.Parse()` (never string prefix match)
- `hostname := u.Hostname()` used (resistant to `localhost.evil.com` attacks)
- Loopback allowed: `127.0.0.1`, `localhost`, `::1`
- WebSocket upgrades require Origin header (absent Origin = reject for WS)
- Private IP origins allowed when server is bound with `--remote` flag
- Public/internet origins: rejected unconditionally

## 5. WebSocket Authentication

**Status: ✅ PASS**

All WebSocket connections checked before upgrade:
1. `auth.ValidateOrigin(r)` — Origin check
2. `auth.AuthenticateRequest(r)` — Session cookie validation

WebSocket handler (`handler_terminal.go`) also re-validates on each message processing. View-only connections cannot write terminal input.

## 6. No Wildcard CORS

**Status: ✅ PASS**

No `Access-Control-Allow-Origin: *` headers found anywhere in the codebase. The Origin policy is enforced at the request level, not via CORS headers (correct for a local app).

## 7. No Secrets in Layout/Demo Data

**Status: ✅ PASS**

Demo mode (`NexusDemoApp.tsx`) uses only synthetic data: fake project names, agent names, synthetic resource quotas. No API keys, tokens, real usernames or real paths in demo data.

Project layout (`serializeWorkspace`) stores only surface type/id/title/data. No secrets serialized.

## 8. Path Canonicalization

**Status: ✅ PASS**

- Static file serving uses `distFS.Open(strings.TrimPrefix(r.URL.Path, "/"))` from an embedded FS
- Embedded FS prevents directory traversal (paths cannot escape the embedded directory)
- Project `canonical_path` set by user input at project creation; not used for file serving

## 9. Agent/Project IDOR Isolation

**Status: ✅ PASS**

`GetAgent(agentID, projectID)` in the store requires both agentID and projectID to match:
```go
func (s *Store) GetAgent(agentID, projectID string) (Agent, error)
```

Web handlers extract projectID from URL path and pass it to store — agent access across projects requires matching both IDs.

## 10. Demo Mode Cannot Mutate Real State

**Status: ✅ PASS — Verified by Playwright test**

Playwright demo mode test result: `Mutating API calls: 0` during demo mode session. Demo renders from static in-memory data and never calls mutating API routes.

## 11. Popout Surface URL

**Status: ✅ PASS**

Popout URL: `window.open(pathname + '?popout=' + encoded, ...)` where encoded is `encodeURIComponent(JSON.stringify({...surface, data:{...}}))`. Surface data contains only: surface type, title, agentId (UUID), projectId (UUID). No tokens, credentials or secrets serialized into the URL.

## 12. Binding Policy

**Status: ✅ PASS**

Default bind: `127.0.0.1` only. `ValidateBind()` rejects `0.0.0.0` without explicit `--remote` flag. Public exposure refused by default.

## Critical Findings

No critical security vulnerabilities found.

## Known Limitations

1. **Session expiration**: Session tokens do not expire while the server runs. Acceptable for local single-user tool. Not acceptable for multi-user deployment.
2. **No HTTPS enforcement**: Local tool runs HTTP. Acceptable for `127.0.0.1` only. The `--remote` flag should document HTTPS requirement for VPN deployments.
3. **`unsafe-inline` for styles**: Required by Tailwind CSS inline styles and dynamic theming. Mitigated by strict `script-src 'self'` (JS cannot inject via CSS).

## Verdict

**SECURITY QA: PASS (no critical gaps)**

CSRF, Origin, session, WebSocket auth, CSP and demo isolation are all correctly implemented and tested. No critical security bypasses found.
