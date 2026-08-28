# AI Control Runtime Validation & Orchestrated Hardening — Design Specification

## 1. Executive Overview

This specification establishes the architectural contract for finalizing and hardening the **AI Control Runtime Subsystem** (`feat/ai-control-runtime-validation`).
The goal is to turn the current implementation into a truthful, robust, deadlock-free, leak-proof, and resilient local control plane for AI coding CLIs.

---

## 2. Goals & Non-Goals

### Goals
1. **Deadlock-Free Host Concurrency**: Eliminate re-entrant mutex locks (`CmdInput` deadlock) and separate state, writer lease, and client broadcasting locks.
2. **True Slash Command Interception**: Implement an instant prefix state machine in the input pipeline so `/ai <cmd>` never leaks to the child provider process stdin, while maintaining zero latency for regular typing and supporting `//ai` escape.
3. **Non-Blocking Fanout & Observer Isolation**: Buffer terminal outputs per-client so slow or hung observers (e.g. TUI or status probe) never block the child process or active writer.
4. **Formal Single-Writer Lease**: Explicit writer token/lease management allowing multiple simultaneous read-only observers and safe handoffs between writers.
5. **Real Windows ConPTY & Truthful Capabilities**: Native ConPTY integration on Windows (or honest downgraded `PARTIAL` reporting if unavailable) and Named Pipe IPC E2E validation.
6. **Verified Account & Context Handoff**: Strict transactional state machine with preflight verification, abort-on-checkpoint-failure, verified provider session continuity, and safe rollback.
7. **Unified RuntimeLauncher**: Single shared launcher component for `ai control start`, account handoff, and context handoff.
8. **PID Recycling Protection**: Identity validation (PID + start time + host generation) before any termination fallback.
9. **Universal Platform Truth**: Real OS, architecture, Go runtime, and build metadata reported across doctor, version, and issue reports.

### Non-Goals
- Adding new provider CLIs.
- Building a web dashboard, cloud sync service, or autonomous task graph.
- Rewriting working protocol components without architectural defects.
- Breaking backwards compatibility with classic execution mode (`ai codex`, `ai claude`, etc.).

---

## 3. Architecture & Subsystems

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                            AI CLI Control Plane                             │
├──────────────────────────────────────┬──────────────────────────────────────┤
│ Interactive Clients / TUI / Slash    │ Multi-Observer Listeners / Telemetry │
└──────────────────┬───────────────────┴───────────────────┬──────────────────┘
                   │                                       │
                   ▼                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            IPC Layer (Client / Protocol)                    │
│    - Unix Domain Sockets ($XDG_RUNTIME_DIR, 0600 umask 0177)                │
│    - Windows Named Pipes (\\.\pipe\ai-control-<id>, Owner SDDL)             │
│    - Versioned Frame Serialization (RPC, Raw Stream, Events)                │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                             SessionHost Daemon                              │
│  ┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────┐  │
│  │ Single-Writer Lease   │ │ Slash Prefix Router   │ │ Bounded Fanout    │  │
│  │ - Owner Conn Tracking │ │ - State Machine       │ │ - Per-Client Queues│ │
│  │ - Graceful Release    │ │ - Instant Passthrough │ │ - Slow-Drop Policy│  │
│  └───────────────────────┘ └───────────────────────┘ └───────────────────┘  │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Terminal Backend Abstraction                                          │  │
│  │ - Unix: creack/pty (PTY / Winsize / RawMode)                          │  │
│  │ - Windows: winpty/ConPTY (Virtual Console / Resize)                   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                 Supervised Child Process (Codex / Claude / AGY / etc.)      │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Key Architectural Patterns & Contracts

### 4.1 Locking & Concurrency Model
To eliminate deadlocks:
- **`stateMu sync.RWMutex`**: Protects `session` metadata, lifecycle states, and generation counters. No I/O or downstream locks acquired under `stateMu`.
- **`clientsMu sync.RWMutex`**: Protects connected clients map and their dedicated send channels.
- **`writerMu sync.Mutex`**: Protects active writer lease connection and lease acquisition/release.
- **Lock Ordering**: If multiple locks are required, the strict order is `writerMu` -> `stateMu` -> `clientsMu`. Never acquire in reverse. Never hold locks during network/terminal I/O.

### 4.2 Slash Command Prefix State Machine
- States:
  - `STATE_IDLE` (start of line / after Enter/Newline)
  - `STATE_SLASH` (received `/` at start of line)
  - `STATE_SLASH_A` (received `/a`)
  - `STATE_SLASH_AI` (received `/ai`)
  - `STATE_CONTROL_CMD` (received `/ai ` or `/ai\r` -> buffered for interception)
  - `STATE_PASSTHROUGH` (diverged from `/ai` prefix -> flushed buffered bytes directly to child process, then continuous streaming)
  - `STATE_ESCAPE` (received `//ai` -> unescapes to `/ai` and forwards to child process)

### 4.3 Bounded Fanout for Multi-Client Broadcasting
- Each attached client connection receives its own bounded ring channel (`chan []byte`, cap = 256 chunks).
- Dedicated worker goroutine pumps chunks from channel to `net.Conn.Write`.
- If client channel is full (slow consumer), drop oldest non-critical terminal chunks and record telemetry/warning, rather than blocking the child terminal reader.

### 4.4 Account & Context Handoff Safety Machine
- Transaction States:
  `REQUESTED` -> `PREFLIGHT` -> `TARGET_VALIDATED` -> `CHECKPOINTED` -> `SOURCE_STOPPED` -> `TARGET_STARTING` -> `TARGET_RUNNING` -> `VERIFYING` -> `VERIFIED` -> `COMPLETED`.
- Failure Paths:
  `FAILED_SAFE` (no side effects), `ROLLBACK_REQUIRED` -> `ROLLING_BACK` -> `ROLLED_BACK` (source restored), or `FAILED_UNSAFE` (explicit error report).
- Checkpoint persistence is mandatory: If `SaveCheckpoint()` fails, transaction **aborts** immediately at `PREFLIGHT` with `FAILED_SAFE`.

---

## 5. Milestone Gates

- **Gate M1 (Input Safety)**: CmdInput deadlock eliminated; prefix state machine stops `/ai` from reaching child stdin; single-writer lease strictly enforced.
- **Gate M2 (Runtime & Fanout)**: Bounded per-client fanout queues; non-blocking broadcast; unified `RuntimeLauncher`; PID recycling validation (`IsProcessAliveWithGeneration`).
- **Gate M3 (Windows)**: Real ConPTY abstraction / truthful capabilities; Named Pipe E2E validation; clean ASCII rendering.
- **Gate M4 (Handoff & Verification)**: Checkpoint mandatory abort on error; verified source stop; resume continuity verification; verified rollback execution.
- **Gate M5 (Truthful Capabilities & Doctor)**: Effective capabilities derived from evidence; Codex/OpenCode truthful in TERMINAL mode; `ai control doctor` output verified.
- **Gate M6 (Context & Redaction)**: Universal secret redaction across checkpoints, lineage, and logs; bounded git status/diff stat limits.
- **Gate M7 (QA & Adversarial Validation)**: Race detector clean across all packages; multiplatform builds (`windows`, `darwin`, `linux`); stress testing and regression suites.

