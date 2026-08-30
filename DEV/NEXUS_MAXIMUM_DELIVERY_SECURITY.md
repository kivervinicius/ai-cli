# IAPro Nexus — Security Deep Review

## Closed findings

### Browser session rotation

Previous failure mode: old browser session could be revoked before replacement entropy was generated. A CSPRNG failure could lock the browser out.

Fix: replacement session ID/CSRF are generated first; current session is only removed after entropy succeeds. Regression test covers entropy failure.

### Session lifecycle routes

Added authenticated + CSRF-protected session rotation/logout routes. Frontend updates both legacy and Nexus CSRF state after rotation and clears/reloads after logout.

### Unauthenticated shell

Frontend no longer renders the Workspace after `authenticated:false`; it displays an explicit bootstrap-required state instead of causing a 401 cascade.

### Origin policy

Origin must match the exact request Host/port and target localhost/loopback/private address. The policy does **not** trust an unrelated RFC1918 origin simply because it is private.

### Bind policy

Wildcard/public binds are rejected by the local control server policy; explicit private remote mode remains constrained and host-filesystem operations are disabled outside loopback.

### WorkPlan concurrency

SQLite uses compare-and-swap on `current_revision`; stale plan edits/suggestions receive conflict rather than silently overwriting approved work.

### Mission lease concurrency

A live mission lease cannot be reacquired while unexpired, even by the same logical runner owner. Fencing tokens remain meaningful.

### Git checkout

Branch is supplied as an argv item, not shell interpolation, and values beginning with `-` / invalid metacharacters are rejected. Checkout is refused when an Agent is actively using the canonical project tree.

## Secret handling

Intelligence configuration stores an environment-variable name or secret-file path, not a raw API key. Runtime API models redact executable/env fields and existing E2E tests assert known secret values do not leak.

## AutonomyContract policy guard — important limitation

Mission execution creates PATH shims for common destructive Git, deployment, external-network, secret-manager and paid-service CLIs. This is an effective policy guard for normal coding-agent command invocation, but it is **not an OS security sandbox**. A sufficiently adversarial process could use an absolute binary path, language network library, direct syscalls or an unwrapped tool.

Therefore:

- `AllowExternalNetwork=false`, `AllowSecretAccess=false`, etc. are enforced product policies in the normal execution path.
- They must not be marketed as a hostile-code containment boundary.
- Strong hostile-code isolation requires OS/container/VM sandbox enforcement in a future hardening layer.

## Remaining production security gates

- full Go 1.25 Web/SQLite/WS suite;
- dependency/SBOM/vulnerability scan on a networked build runner;
- final CI origin/CSRF/WS auth E2E;
- physical Windows/macOS runtime security smoke;
- optional external penetration review before public remote-mode exposure.
