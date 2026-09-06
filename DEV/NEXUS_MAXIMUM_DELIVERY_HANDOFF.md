# IAPro Nexus — Maximum Delivery Handoff

## Candidate

- Version: `0.5.0-beta.5`
- Branch: `feat/nexus-maximum-delivery`
- Product model: local Web-first Workspace OS with Direct -> Autonomous progression.

## Non-negotiable architecture retained

Direct AI sessions do not require Intelligence, Maestro, WorkPlan, Scheduler auto-selection or Mission. Automation augments rather than removes manual control.

## Blocks 1–5 status

### 1 — Mission Runner

Implemented real PackageExecutor contracts, staged retry/remediation, evidence, DAG/parallel release, independent review requirements and final verification semantics.

### 2 — Durability

SQLite RunRepository contracts, PromptVersion/ExecutionSnapshot linkage, lease TTL/heartbeat/fencing/reclaim and schedule claim semantics are present.

### 3 — Take Control

Requires allocated Agent, pauses orchestration, records manual workspace checkpoint/diff evidence, and resumes through testing instead of blindly repeating implementation.

### 4 — WorkPlan

Structural editing, DAG validation, locks, acceptance criteria, AI suggestion Apply/Reject and optimistic concurrency are connected to backend persistence.

### 5 — E2E

Frontend/backend route contracts were rechecked and corrected. Hermetic tests cover product contracts; `scripts/nexus-e2e-local.go` is the real-provider local smoke gate.

### Attention regression follow-up

The proxy-nginx AGY question was reproduced in the frontend. Runtime responses now route through an explicit external-input path, answered prompts reset detector history and persisted attention metadata, and the attention UI no longer consumes a Workspace grid row. Focused Go/frontend validation is green; native platform, same-SHA CI, and real AGY trust/E2E evidence remain open.

## Next promotion action

Run `DEV/NEXUS_MAXIMUM_DELIVERY_LOCAL_VALIDATION_PROMPT.md` on Linux, macOS and Windows / CI with Go 1.25 and at least one real authenticated provider. Attach the resulting reports. Only promote to production `GO` if all mandatory gates pass.

## 2026-08-30 continuation

The finalization pass fixed the objectively reproduced stdout pipe deadlock in the test capture harness, moved the web workspace model to v2 with migration/focused-stack/hydration protections, and isolated terminal control frames from provider output. Local Linux/frontend/race/cross-build evidence is green. Native platform, same-SHA CI, release snapshot, real provider and automation gates remain open; current recommendation is `NO_GO` until those are executed and reviewed.
