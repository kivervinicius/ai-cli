# IAPro Nexus — Canonical Product Contract

Date: 2026-09-01  
Status: **AUTHORITATIVE FOR V0/V1 REBUILD**

This document supersedes product-language conflicts in the 2026-08-29 Maximum Delivery and Workspace OS specs. Existing backend/storage contracts remain compatible unless explicitly replaced below.

## 1. Product identity

IAPro Nexus is a **local-first, project-centric Agentic Product Development Workspace / Workspace OS**.

It is not:
- a generic workflow automation product;
- a full IDE;
- a full Git client;
- a Mission-only runner;
- a mandatory Maestro shell;
- a separate "Nexus Intelligence" product brain.

## 2. Primary user model

A Project opens directly into its Workspace.

First-class Workspace surfaces:
1. persistent Agent;
2. normal Project Shell;
3. project/filesystem tools;
4. Composer, containing its Flow Draft;
5. active Flow Run.

Direct Agent work and Project Shell are always available. Composer, Flow, Maestro, scheduler and automation are optional layers, never mandatory gateways to direct provider work.

## 3. Identity and runtime invariants

`Agent != RuntimeGeneration != Provider Session != Workspace View`.

- AgentID is durable identity.
- RuntimeGeneration is replaceable execution state.
- Provider session/resume is provider-specific continuity.
- View is presentation only.
- The same logical View may appear as tab, split or Desktop window.
- Closing an Agent View never stops the Agent.
- Closing a Project Shell View stops only that shell runtime.

## 4. Workspace presentation

Presentation modes:
- `TABS`: stacks/splits;
- `DESKTOP`: movable/resizable/minimizable/maximizable windows.

Both modes reference the same stable `viewId` / `logicalKey`. Changing presentation cannot recreate Agents, shells, sessions or Flow Runs.

`+ New` / command launcher must expose at minimum:
- Agent / AI Session;
- Project Shell;
- Composer;
- active Flow Run;
- existing Project/filesystem tools.

## 5. Direct work

### Ask existing Agent

`Ask` always targets a supplied AgentID.

- Running Agent: submit work to the existing Agent/runtime.
- Stopped/recoverable Agent: explicit Start & Ask may start/recover a RuntimeGeneration.
- Ask must never create a new Agent as an implicit side effect.

### New AI Session

A user may create a new Agent/session explicitly without using Composer or Flow.

### Project Shell

A Project Shell:
- contains no AI semantics;
- starts in canonical Project cwd;
- reuses the supervised PTY/ConPTY terminal infrastructure;
- has lifecycle independent from Agent lifecycle.

## 6. Composer

Composer is the intelligent editing/planning environment. It evolves the existing Work/PlanBuilder functionality instead of creating a competing product.

User-facing `Direct / Assisted / Planned` and `DIRECT / ASSISTED / GUIDED / ORCHESTRATED / AUTONOMOUS` are **not** primary UX modes.

Composer may internally use existing Intelligence CLI/API providers, but Intelligence is an implementation capability, not a separate user-facing product brain.

For simple work, Composer may finish as:
- Send to existing Agent;
- New AI Session;
- Copy Prompt;
- Turn into Flow.

Flow is optional.

## 7. Context Readiness and Maestro boundary

Context Readiness states:
- `MISSING`
- `HYDRATING`
- `READY`
- `STALE`
- `FAILED`

Composer planning/refinement requires `READY`. Direct Agent work and Project Shell do not.

Fingerprint includes bounded operational identity such as:
- project;
- branch;
- HEAD;
- dirty fingerprint;
- Maestro version/status.

Maestro remains an independent integration. Nexus may detect, inspect, update, list real skills, request supported operations and record readiness status. Nexus must not invent Maestro skills, duplicate Maestro semantic memory, or fabricate hydration success.

Canonical durable project knowledge remains `AGENTS.md` and `DEV/` artifacts where present.

## 8. Flow façade

User-facing mappings:
- `WorkPlan` -> `FlowDefinition`
- `WorkPackage` -> `FlowStep`
- `PlanRevision` -> `FlowRevision`
- `MissionRun` -> `FlowRun`

These are compatibility façades. Do not migrate the database only to rename concepts, and do not create a second planner/runner.

V0 Flow graph supports only:
- sequence;
- dependency;
- parallel group.

Not V0:
- cron/webhook generic automation nodes;
- arbitrary scripts as graph nodes;
- runtime graph mutation;
- dynamic loops;
- n8n-like generic canvas.

A Flow Draft is side-effect-free. No Agent creation, provider allocation, runtime launch or MissionRun occurs until explicit **Approve & Run**.

## 9. Flow Step assignment

Step assignment strategy:
- `EXISTING`: explicit persistent Agent;
- `CREATE`: create a specialist Agent only after approval/execution preparation;
- `AUTO`: choose compatible existing Agent/resource first, create only when necessary.

Existing Resource Scheduler, health, quota and manual overrides remain authoritative. Quota/token/cost data must never be fabricated.

## 10. Context Capsule and Work Receipt

Every dispatched Flow Step receives a bounded typed Context Capsule, not the whole repository or raw conversation transcript.

Context Capsule contains only relevant data, such as:
- project/branch/HEAD/dirty fingerprint;
- Flow revision and Step;
- relevant durable context;
- relevant paths;
- direct dependency Work Receipts;
- real Maestro skills;
- acceptance criteria;
- constraints.

Every completed/failed Step produces a factual typed Work Receipt containing evidence that actually exists, such as:
- summary;
- changed files;
- commands;
- tests/verification;
- decisions;
- artifacts;
- remaining issues;
- AgentID;
- base/result revisions;
- timestamps.

Missing evidence stays empty/unknown. Never synthesize success evidence.

Dependents receive receipts, not raw upstream transcripts.

## 11. Flow execution

Reuse the existing durable Mission Runner.

The runner must preserve:
- immutable ExecutionSnapshot;
- dependency DAG;
- parallel execution;
- leases/fencing/heartbeats;
- persisted dispatch intent before provider launch;
- restart recovery without silent double dispatch;
- bounded retries/remediation;
- real verification;
- independent review where required and available;
- pause/resume/cancel;
- Take Control / Return Control;
- autonomy boundaries.

User-facing FlowRun states map fail-closed to:
- `QUEUED`
- `READY`
- `RUNNING`
- `VERIFYING`
- `COMPLETED`
- `BLOCKED`
- `FAILED`
- `CANCELED`

Unknown internal states never map to `COMPLETED`.

Example dependency contract:

`A -> (B || C) -> D`

- A executes first.
- B and C become runnable only after A is verified.
- D becomes runnable only after B and C are verified.
- If B/C fail or block, D is not dispatched.

## 12. Autonomy concepts

Flow policy:
- `GUIDED`
- `AUTONOMOUS`

This is separate from provider-specific permission/execution mode such as safe/plan/yolo/custom.

GUIDED pauses on decisions outside approved scope or safety/verification policy. AUTONOMOUS proceeds only inside approved Flow, security, resource and verification boundaries.

## 13. Verification principle

`GOAL -> EXECUTE -> VERIFY -> REVIEW -> STOP`.

Provider output saying "done" is not completion evidence.

A reviewer must have a real independent identity. Reviewer unavailability is an evidence gap or blocked/review failure according to policy; it is never implicit approval.

## 14. Capability preservation gate

No useful baseline capability may silently regress:
- project create/register;
- filesystem browse/scan/inspect/mkdir;
- OS file manager/editor/external terminal actions;
- Branch display/switch;
- providers/profiles/accounts/auth/capabilities/config;
- usage/quota/reset/cache semantics;
- resource discovery/recommendation/manual allocation;
- persistent Agents/config/Safe Apply/isolation/worktree;
- runtime generations/reconnect/continuity;
- terminal ANSI/input/output/resize/control;
- Windows ConPTY/named pipes;
- Intelligence CLI/API implementation capability;
- WorkPlan revisions/diff/restore/clarifications/dependencies/parallelism;
- Mission durability/verification/Take Control;
- schedules;
- Maestro integration;
- updates;
- CSRF/origin/secrets/isolation/launch-env security.

## 15. Cross-platform and release truth

Linux, macOS and Windows are first-class targets.

A production GO requires same-SHA evidence for applicable platform tests, browser E2E, security and release gates. Cross-compilation is not native runtime proof. Historical evidence from another SHA is not current evidence.
