# IAPro Nexus V1 — Autonomy & Mission Runner Report

## 1. Overview
The Nexus Autonomous Mission Runner (`internal/nexus/runner`) delivers a production-grade, state-machine driven execution model with bounded retries, independent review loops, and strict verification gates under an explicit `AutonomyContract`.

## 2. State Machine Architecture
```
[READY] ➔ [ALLOCATING] ➔ [COMPILING] ➔ [EXECUTING] ➔ [TESTING]
                                            ▲             │
                                            │             ▼
                                     [REMEDIATING] ◄── [REVIEWING] ➔ [VERIFIED] ➔ [COMPLETED]
                                            │
                                            ▼ (Max retries exceeded)
                                     [ESCALATED / FAILED]
```

## 3. Autonomy Contract
- **Bounded Retries**: Strictly enforces `MaxRetries` (default: 3) to prevent infinite remediation loops and token exhaustion.
- **Independent Verification**: Mandatory execution of deterministic test commands (`go test -race ./...`, test runners) before package verification.
- **Destructive Git Protection**: Blocks destructive git checkouts when agents are actively running in canonical project workspaces.
- **Heartbeat & Lease Continuity**: `LeaseManager` actively monitors run health, eliminating orphaned processes.

## 4. Verification Evidence
- Unit tests: `internal/nexus/runner/runner_test.go` (100% pass under `-race`).
- Integration tests: End-to-end plan step advancement via `/api/v1/runs/:id`.
