# AI CLI — Final Security Report

**Date:** 2026-08-28 · **Branch:** `fix/control-production-readiness`

## Web transport

| Control | Status | Evidence |
|---|---|---|
| Loopback default (`127.0.0.1`) | ✅ | `server.go` default + live bind |
| IPv6 loopback `::1` | ✅ | `ValidateBind` accepts `::1` |
| Public bind refused (not warned) | ✅ | live: `--listen 8.8.8.8` → hard error |
| Private bind requires `--remote` | ✅ | live: `--listen 192.168.1.5` → hard error |
| CGNAT `100.64.0.0/10` treated non-public | ✅ | `web/bind.go` `isPrivateIP` + tests |
| Hostname bind refused | ✅ | `ValidateBind` rejects non-IP |

## Auth & session

| Control | Status | Evidence |
|---|---|---|
| Cryptographic one-time bootstrap token | ✅ | 32-byte hex, single exchange |
| Authenticated session → HttpOnly cookie | ✅ | `HttpOnly: true` |
| SameSite=Strict | ✅ | |
| Session expiration / rotation / logout | 🟡 | sessions are in-memory with bootstrapping; expiry/rotation not fully exercised this pass |
| Secure cookie under TLS | 🟡 | loopback-only (no TLS termination); documented SSH-tunnel model |

## Request hardening

| Control | Status | Evidence |
|---|---|---|
| CSRF on state-changing REST | ✅ | `X-CSRF-Token` verified against session |
| Origin validation (incl. WS) | ✅ | `auth.CheckOrigin` loopback-only |
| WebSocket: session + Origin + runtime auth + writer lease | ✅ | `routeRuntime` checks |
| No wildcard CORS | ✅ | no CORS headers set at all |
| CSP + XCTO + Referrer + Permissions + frame-ancestors | ✅ | live curl verified |
| XSS: metadata not rendered as trusted HTML | ✅ | xterm writes output as terminal data, not raw HTML |

## IPC / process

| Control | Status | Evidence |
|---|---|---|
| Bounded frame reads (no unbounded alloc) | ✅ | `readBoundedLine` + fuzz |
| Oversized frame rejected with visible error | ✅ | unit + fuzz |
| Malformed/partial frame + connection close | ✅ | fuzz + lifecycle tests |
| Protocol version enforcement | ✅ | `ERROR_PROTOCOL_VERSION` + test |
| PID reuse protection (generation token) | ✅ | `IsProcessAliveWithGeneration` |
| Named Pipe owner-restricted SDDL | ✅ | `endpoint_windows.go` |

## Data

| Control | Status | Evidence |
|---|---|---|
| Redaction: OpenAI/Anthropic/Google/GitHub/AWS/JWT/Bearer/PEM/cookies/auth/.env | ✅ | extended patterns + 13-case table test |
| Redaction applied to checkpoints/kickoff prompts | ✅ | `FormatKickoffPrompt` + `CaptureWorkCheckpoint` |
| Chain-of-thought never transferred | ✅ | context handoff transfers bounded work checkpoint only |
| Secret leakage E2E (runtimes API) | ✅ | `e2e_test.go` |
| Fuzz redaction (149k execs) | ✅ | no panics, no marker leakage |

## Terminology (docs)

- README no longer uses "hermetic sandbox" for HOME isolation; uses **credential
  isolation / isolated profiles** (matches code reality: same-user process, isolated
  config HOME).

## Out of scope (by charter)

- Public TLS/identity serving (refused); remote access is SSH tunnel / private VPN.
- Distributed multi-node control plane (interfaces only).

## Residual P2 items

- Web session expiration/rotation not runtime-exercised (in-memory session store).
- CSP `connect-src ws: wss:` is permissive-by-scheme (mitigated by loopback-only bind
  + Origin enforcement).
