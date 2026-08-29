# IAPro Nexus V1 — Security & Hardening Report

## 1. Network & Bind Security
- **Strict Bind Validation**: Wildcard addresses (`0.0.0.0`, `::`) are rejected. Binding requires explicit loopback (`127.0.0.1`, `::1`) or explicit private/VPN IP addresses (`10.x`, `192.168.x`, `100.64.x`) under `--remote`.
- **Public IPs Refused**: Binding to public interfaces is blocked.

## 2. Origin & Session Hardening
- **Strict Origin Checking**: Rejects foreign origins and DNS rebinding attacks.
- **Session Lifecycle**: 24-hour absolute expiration with 4-hour idle timeout.
- **Session Rotation & Revocation**: Atomic session rotation and revocation (`RevokeSession`).
- **CSRF & WebSocket Auth**: Strict origin and CSRF header validation on all mutating REST endpoints and WebSocket upgrades.

## 3. Terminal Authority & Isolation
- **Single-Writer Lease Authority**: One writer lease authority on `AgentTerminalBroker` tied to stable `AgentID`. Split authority eliminated.
- **Git Branch Safety Policy**: Direct project checkout is blocked if agents are actively executing in the canonical project tree.
