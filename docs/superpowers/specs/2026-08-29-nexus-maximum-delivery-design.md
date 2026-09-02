> **Supersession notice (2026-09-01):** Product-language and UX conflicts in this historical spec are superseded by `docs/superpowers/specs/2026-09-01-nexus-canonical-product-contract.md`. Backend preservation rules and non-conflicting architecture remain valid.

# IAPro Nexus Maximum Delivery Design

## Product intent
IAPro Nexus is a local-first AI engineering workspace that must preserve the freedom of the original ai-cli while adding Project-first organization, Workspace OS, persistent agents, optional Nexus Intelligence, optional Maestro guidance, structured WorkPlans, and autonomous Missions.

## Non-negotiable product rule
No subsystem may become a mandatory gateway to another subsystem unless required by safety or technical necessity. Direct provider work must remain first-class. Mission, Maestro, Intelligence, and Scheduler are optional layers of increasing automation.

## Execution modes
- DIRECT: user chooses provider/profile/session and works immediately.
- ASSISTED: direct work plus optional Nexus Intelligence context/prompt assistance.
- GUIDED: assisted work plus optional Maestro process/skill recommendations.
- ORCHESTRATED: structured WorkPlan with user-controlled assignments and orchestration.
- AUTONOMOUS: approved WorkPlan + AutonomyContract + durable MissionRunner.

## Core ownership
- Project owns Persistent Agents, Sessions, WorkPlans, Missions, layouts and policy.
- AgentID is stable; RuntimeGeneration is ephemeral.
- ai-cli provider/session/account/quota/handoff semantics are the execution-layer source of truth.
- Nexus Intelligence owns semantic interpretation and clarification, not Maestro skills.
- Maestro owns process/skills/risk/review/verification and may be OFF or DEGRADED without blocking Direct work.
- Resource Scheduler chooses resources only when policy requests recommendation/auto allocation.
- MissionRunner executes approved WorkPackages durably and must survive Nexus/browser restarts.

## Terminal invariants
- Terminal surface is addressed by AgentID, not RuntimeID.
- Exactly one writer authority exists per Agent.
- RuntimeGeneration replacement must not require the user to recreate the terminal surface.
- Same-provider native resume and cross-provider Context Handoff are distinct and truthfully labeled.

## WorkPlan model
Mission -> Phase -> WorkPackage -> Task. Dependencies form a DAG. Priority never bypasses dependencies. PlanRevision is source of truth. PromptVersion is compiled from a PlanRevision. Running work binds to immutable ExecutionSnapshot.

## Autonomous execution
MissionRunner must: persist MissionRun/PackageRun state, lease ownership, heartbeat, dispatch real agents, enforce dependencies and parallelism, run verification, invoke real independent review when required, remediate failures, detect no-progress, replan within approved scope, and escalate only decisions outside AutonomyContract.

## Cross-platform
Linux: PTY + UDS. macOS: PTY + UDS. Windows: ConPTY + Named Pipe. CI and release must prove compile/test on all three; runtime smoke tests must exist per platform where supported.

## Release invariant
A release is valid only when git status is clean, HEAD matches embedded binary commit, VERSION matches binary version, frontend and Go verification pass against that exact HEAD, and reports record the exact SHA.
