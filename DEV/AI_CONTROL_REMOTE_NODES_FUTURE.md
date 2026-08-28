# AI Control Remote Nodes & Multi-Machine Architecture (Future Blueprint)

## 1. Vision & Abstract
The future vision of AI Control is a distributed, multi-machine control plane where a single developer or team can supervise agents running across multiple physical or virtual machines (e.g. desktop workstation with GPUs, cloud VMs, edge devices) from a single unified browser or TUI interface.

---

## 2. Target Distributed Topology

```text
┌──────────────────────────────────────────────────────────────┐
│                  Developer Interface                         │
│       Browser / Web Control Center / TUI / Mobile            │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTPS / WSS / mTLS
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                    AI Control Hub Node                       │
│    - Global Session Index & Topology Catalog                 │
│    - Machine Registry (MachineID, Location, Capabilities)    │
│    - Terminal Stream Multiplexer                             │
└──────────────┬───────────────────────────────┬───────────────┘
               │ mTLS Control Tunnel           │ mTLS Control Tunnel
               ▼                               ▼
┌──────────────────────────────┐ ┌──────────────────────────────┐
│       Node A (Local / Dev)   │ │    Node B (Cloud / GPU VM)   │
│  - Runtime Launcher          │ │  - Runtime Launcher          │
│  - SessionHost (Codex / AGY) │ │  - SessionHost (Claude/GPU)  │
│  - PTY / ConPTY              │ │  - PTY / ConPTY              │
└──────────────────────────────┘ └──────────────────────────────┘
```

---

## 3. Protocol Elements

### 3.1 Mutual TLS (mTLS) & Enrollment
- Each node generates a local ed25519 keypair and requests enrollment with the Hub via one-time pairing code (`ai control node pair <code>`).
- Hub issues node client certificates validating the `MachineID`.

### 3.2 Capability Negotiation
- Remote nodes broadcast:
  - Installed providers and runtime versions (`codex --version`, `claude --version`).
  - Terminal engines (Linux PTY vs Windows ConPTY).
  - Hardware resources (CPU cores, memory, GPU availability).

### 3.3 Terminal Stream Transport
- Low-latency multiplexed streams (e.g. gRPC streams or WebTransport over QUIC) forwarding ANSI sequences, window resize events, and writer leases.

---

## 4. Current Compatibility Model
The current codebase (`ai-cli v0.4.0`) is explicitly designed with clean boundaries for this future architecture:
- `registry.RuntimeSession` contains `MachineID`, `Location`, and `Transport`.
- Terminal WebSockets communicate via versioned JSON messages (`output`, `input`, `resize`, `lease_acquire`, `lease_release`).
- State is decoupled from the UI layer, enabling remote node proxies without rewriting the Web or TUI clients.
