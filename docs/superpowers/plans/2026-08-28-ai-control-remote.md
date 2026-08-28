# AI Control Remote Private Access — Implementation Plan (Subproject C)

## Overview
This plan implements the private remote access layer and future multi-machine boundaries.

---

## Tasks

### Task C1: Machine Identity & Future-Ready Metadata
- **Files**:
  - `internal/control/registry/models.go`
  - `internal/control/registry/machine.go`
- **Details**:
  - Add `MachineID`, `Location`, and `Transport` to `RuntimeSession`.
  - Implement deterministic `LocalMachineID()` using hostname / `/etc/machine-id`.
- **Verification**: `go test -race -v ./internal/control/registry/...`

### Task C2: Private Network Binding Validation & Warning
- **Files**:
  - `internal/control/web/server.go`
  - `internal/control/web/auth.go`
- **Details**:
  - Validate non-loopback IP bindings against private network ranges (RFC 1918 / CGNAT).
  - Print explicit security notices and ensure strict token bootstrap.
- **Verification**: `go test -race -v ./internal/control/web/...`

### Task C3: Architectural Blueprint & Deferred Log
- **Files**:
  - `DEV/AI_CONTROL_REMOTE_NODES_FUTURE.md`
  - `DEV/AI_CONTROL_DEFERRED.md`
- **Details**:
  - Document future multi-node topology, mTLS handshakes, and remote terminal multiplexing.
  - Document deferred cloud / public-hosting backlog.
- **Verification**: Verify documentation formatting and completeness.

### Task C4: SSH Tunnel Simulation Test
- **Files**:
  - `internal/control/web/remote_test.go`
- **Details**:
  - Test simulated port-forwarding proxy connection verifying token exchange, cookie persistence, and WebSocket streams.
- **Verification**: `go test -race -v -run TestRemote_SSHTunnel ./internal/control/web/...`

