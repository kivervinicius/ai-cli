# AI Control Remote Private Access — Design Specification (Subproject C)

## 1. Executive Overview
Subproject C enables developers to securely access and control their local coding agents from a remote machine (e.g. laptop controlling a high-powered workstation or headless server) via encrypted private channels. It avoids exposing unauthenticated services to the public Internet while laying clean foundations for future multi-node topologies.

---

## 2. Remote Access Paradigms

### 2.1 Paradigm A: SSH Port Forwarding Tunnel (Primary & Recommended)
- The remote server runs:
  ```bash
  ai control web --no-open
  ```
  Output:
  ```text
  URL:       http://127.0.0.1:45837
  Bootstrap: http://127.0.0.1:45837/?token=3f8a...
  ```
- The local laptop runs:
  ```bash
  ssh -N -L 8765:127.0.0.1:45837 user@dev-workstation
  ```
- The developer opens `http://127.0.0.1:8765/?token=3f8a...` in their browser.
- **Security Advantages**:
  - The Web server continues listening **strictly on loopback** (`127.0.0.1`).
  - Encrypted end-to-end with existing OpenSSH keys/certificates.
  - Zero ports open to the external network or LAN.
  - Browser same-origin and WebSocket policies are satisfied natively.

### 2.2 Paradigm B: Private Network Listen (Explicit Opt-In)
- For trusted Tailscale, WireGuard, or corporate overlay networks:
  ```bash
  ai control web --listen 100.64.0.15 --port 8765 --no-open
  ```
- The server checks that the bound IP is private (RFC 1918 or CGNAT 100.64.0.0/10) and warns if an attempt is made to bind public interfaces without reverse proxy.

---

## 3. Distributed Readiness & Model Boundaries
To prepare for future multi-machine nodes without premature distributed complexity:
- `RuntimeSession` model includes:
  - `MachineID string`: Unique hardware identifier of the host (defaults to local hostname or machine UUID).
  - `Location string`: Physical/logical location tag (defaults to `"local"`).
  - `Transport string`: IPC transport mechanism (e.g. `"unix"`, `"named_pipe"`, `"ssh_tunnel"`).

