# Nexus V1 — Security Report & Platform Matrix

## Security (charter §93-96, §138-141, §147)

### Transport & bind
- Loopback default (`127.0.0.1`/`::1`); public bind **refused**; private requires `--remote`; CGNAT `100.64/10` handled. (live-verified prior gate)
- Remote = SSH tunnel / private VPN; no public TLS serving in 1.0.

### Auth & session
- Cryptographic one-time bootstrap token → HttpOnly SameSite=Strict cookie; CSRF on state-changing REST; Origin validation on REST + WS; authenticated WS; no wildcard CORS; restrictive CSP + security headers (live-verified).

### Application
- **IDOR guard:** `store.GetAgent(id, projectID)` enforces project scoping; cross-project reads → not found (tested).
- **Path canonicalization:** `store.CanonicalPath` = Abs → Clean → EvalSymlinks → must exist as directory (rejects `..`/symlink-escape/non-dirs; tested).
- **CSRF** enforced in the web layer (test asserted 403 without token).
- **XSS:** terminal output goes through xterm (not `innerHTML`); provider metadata is never trusted HTML.
- **Secrets:** never persisted in SQLite; redaction pipeline on checkpoints/events/reports; `Env/Args/Binary` excluded from registry JSON.

### Tests covering security
- `TestNexusProjectsAndAgentsAPI` (CSRF 403 path), IDOR guard in store test, canonical-path rejection tests.

## Platform matrix

| Platform | Build | Unit/race | Runtime E2E (CI) | Local runtime | Verdict |
|---|---|---|---|---|---|
| Linux amd64/arm64 | ✅ | ✅ 25 pkgs `-race` | ✅ authored (prior gate) | ✅ live this gate | SUPPORTED |
| Windows amd64/arm64 | ✅ cross | ✅ vet | ✅ authored (ConPTY/NamedPipe/PowerShell) | pending user | CODE-COMPLETE, runtime pending |
| macOS amd64/arm64 | ✅ cross | ✅ vet | ✅ authored (PTY/socket/Web) | pending user | CODE-COMPLETE, runtime pending |

Live this gate (Linux): project/agent API, agent start → runtime RUNNING, agent
terminal WS 101, persistence across web restart.

## Honesty

Windows/macOS are not claimed fully runtime-tested until CI + local confirmation
produce evidence (§148). No capability is reported as SUPPORTED without proof (§158).
