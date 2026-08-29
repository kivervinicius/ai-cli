# IAPro Nexus Maximum Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a release-grade IAPro Nexus that preserves direct ai-cli workflows and connects frontend, backend, runtime, planning and autonomy into one verified product.

**Architecture:** Preserve the existing provider/session/runtime layer and expose it as first-class Direct/Assisted workflows in Workspace OS. Stabilize Agent lifecycle and terminal invariants before layering Intelligence, Maestro, WorkPlan and durable Mission execution. Every layer must remain independently usable and degradation-safe.

**Tech Stack:** Go 1.25, SQLite, React 19, TypeScript 5.9, xterm.js, Tailwind, Vitest, GitHub Actions/GoReleaser.

**Spec:** `docs/superpowers/specs/2026-08-29-nexus-maximum-delivery-design.md`

## Global Constraints
- Direct provider work must never require Mission, WorkPlan, Maestro or Nexus Intelligence.
- AgentID is stable; RuntimeGeneration is ephemeral.
- UNKNOWN quota is never treated as best/100%.
- Cross-provider continuity is Context Handoff, never native resume.
- Maestro unavailable means MAESTRO_DEGRADED; Nexus never invents skills.
- Public bind and wildcard bind are refused.
- No secrets in events, prompts, reports or SQLite plaintext fields intended for observability.
- Release artifacts must be reproducible from a clean HEAD.

---

### Task 1: Reconcile branch functionality and protect direct ai-cli workflows
**Files:** `internal/app/*`, `internal/core/provider/*`, `internal/control/*`, `web/src/features/work/*`, `web/src/nexus/api.ts`
**Produces:** first-class Direct/Assisted provider/session APIs and UI entry points without Mission dependency.
- [ ] Add regression tests that Direct provider launch works with Maestro/Intelligence disabled.
- [ ] Verify test fails on any route that requires Mission.
- [ ] Wire existing provider/profile/session/handoff capabilities into Project Workspace actions.
- [ ] Add UI tests/model tests for creating Direct sessions and promoting work to plan/mission.
- [ ] Run focused Go and frontend tests.

### Task 2: Make Agent terminal continuity truthful and single-authority
**Files:** `internal/control/web/broker.go`, `internal/control/web/handler_terminal.go`, `internal/control/host/*`, `web/src/features/agents/AgentTerminal.tsx`
**Produces:** one writer lease authority per Agent and automatic runtime-generation rebinding/reconnect.
- [ ] Add tests for two viewers, one controller, transfer control and runtime generation switch.
- [ ] Make tests fail against current double-authority behavior.
- [ ] Consolidate writer ownership under AgentTerminalBroker/SessionHost contract with one authoritative lease path.
- [ ] Emit generation-change protocol event and reconnect/rebind xterm automatically by AgentID.
- [ ] Verify bounded replay and no cross-talk.

### Task 3: Finish Agent configuration, recovery and Safe Apply
**Files:** `internal/nexus/nexus.go`, `internal/nexus/config.go`, `internal/control/launcher/launcher.go`, provider drivers, store models/tests.
**Produces:** operational model/isolation/options/env configuration with transactional revision application and rollback.
- [ ] Add failing tests proving model/options/isolation reach provider driver launch arguments/environment.
- [ ] Add failing recovery test proving project canonicalPath and full config are restored.
- [ ] Add failing Safe Apply rollback test.
- [ ] Implement candidate->validate->start new generation->verify->commit or rollback.
- [ ] Record honest continuity status.

### Task 4: Correct resource discovery and scheduler semantics
**Files:** `internal/nexus/resource_discovery.go`, `scheduler.go`, `resource_recommendation.go`, quota model/tests.
**Produces:** capability-gated, health/quota-aware resource recommendation with honest UNKNOWN handling.
- [ ] Add failing UNKNOWN quota test.
- [ ] Add failing required-capability exclusion test.
- [ ] Use actual quota status and hard capability gates before scoring.
- [ ] Expose explanation API and UI evidence.

### Task 5: Make Nexus Intelligence optional but real
**Files:** `internal/nexus/intelligence/*`, app/web handlers, `WorkSurface.tsx`, settings/provider UI.
**Produces:** CLI IntelligenceProvider adapters plus OpenAI-compatible provider, explicit availability and clarification lifecycle.
- [ ] Add provider interface tests for CLI and API providers.
- [ ] Add test: DIRECT works with zero intelligence providers.
- [ ] Add test: ASSISTED/PLANNED require configured provider unless user explicitly chooses deterministic template mode.
- [ ] Generate plan outline from selected IntelligenceProvider instead of unconditional fallback.
- [ ] Persist clarifications/decisions and expose blocking-question UI.

### Task 6: Restore Maestro boundary and degradation behavior
**Files:** `internal/nexus/maestro.go`, handlers/UI/tests.
**Produces:** machine-readable Maestro bridge with no fabricated skills and portable discovery.
- [ ] Add failing test: unavailable Maestro returns DEGRADED with zero invented recommendations.
- [ ] Remove hardcoded user-specific paths and skill names.
- [ ] Keep Direct/Assisted work usable when Maestro is off/unavailable.

### Task 7: Complete WorkPlan, PromptVersion and execution snapshots
**Files:** store migrations/models/plan APIs, `PlanBuilderSurface.tsx`, prompt compiler.
**Produces:** editable DAG plan with revisions, compiled prompts and immutable execution snapshots.
- [ ] Add schema/tests for PromptVersion, dependencies, shared artifacts and ExecutionSnapshot bindings.
- [ ] Add plan dependency validation/cycle test.
- [ ] Add UI operations for reorder/priority/dependency/parallel-group and prompt preview.
- [ ] Ensure active executions never silently mutate when plan revisions change.

### Task 8: Turn MissionRunner into real durable execution
**Files:** `internal/nexus/runner/*`, store migrations/repositories, launcher/agent APIs.
**Produces:** persistent MissionRun/PackageRun DAG executor that launches real Agents.
- [ ] Add failing restart-survival test using SQLite.
- [ ] Add failing test that EXECUTING package creates/attaches a real Agent runtime.
- [ ] Add dependency/parallelism tests.
- [ ] Implement persistent leases, heartbeat TTL/fencing and stale recovery.
- [ ] Implement real verification/review/remediation hooks and no-progress policy.
- [ ] Enforce AutonomyContract before destructive/external actions.

### Task 9: Scheduling and manual takeover/return-to-mission
**Files:** runner/store/web APIs/UI.
**Produces:** Start Now/At/After Mission/When Capacity Available and package-level MANUAL/AUTO transitions.
- [ ] Add scheduling persistence tests.
- [ ] Add Take Control / Return to Mission state tests.
- [ ] Ensure independent packages continue while one awaits user escalation.

### Task 10: Security, accessibility and full front-back integration
**Files:** web server/auth/origin/CSRF/FS handlers, React surfaces, E2E harness.
**Produces:** secure loopback/private access and verified UI flows.
- [ ] Test public/wildcard bind rejection, CSRF, Origin, WS auth, IDOR, traversal/symlink escape and secret redaction.
- [ ] Add integrated browser smoke flows Project -> Direct Agent -> Terminal -> Handoff -> Plan -> Mission.
- [ ] Verify keyboard-only Workspace OS, themes, high contrast and mobile behavior.

### Task 11: Cross-platform and release provenance
**Files:** platform runtime code, CI, GoReleaser, version build scripts, docs.
**Produces:** reproducible release candidate and platform evidence.
- [ ] Run/define Linux PTY+UDS smoke tests.
- [ ] Run/define macOS PTY+UDS CI smoke tests.
- [ ] Run/define Windows ConPTY+Named Pipe CI smoke tests.
- [ ] Make default CLI Web-first and remove/deprecate obsolete TUI UX paths without breaking compatibility aliases.
- [ ] Require clean tree, exact HEAD/version embedding and artifact checksum manifest.
- [ ] Generate final acceptance report with exact commands and SHA.
