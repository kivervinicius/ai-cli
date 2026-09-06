# Nexus V1 — Maestro WIP Salvage

## Source audited

`IAPro-Community/Orquestrador-Maestro` @ `feat/cli-novo-wip` (`8f3763e`), shallow clone.
~5000 lines of planner/interview/graph code plus 614 lines of tests.

## Classification (charter §60-61)

### KEEP_IN_MAESTRO (concepts live in Maestro; Nexus only consumes)

| Component | Files | Verdict |
|---|---|---|
| SemanticPlanner | `runtime/planner/semantic-planner.js` (164) | KEEP_IN_MAESTRO |
| DeterministicFallbackPlanner | `runtime/planner/deterministic-fallback-planner.js` | KEEP_IN_MAESTRO |
| PlanSemanticDiff | `runtime/planner/plan-semantic-diff.js` + test | KEEP_IN_MAESTRO |
| MissionBriefBuilder | `runtime/planner/mission-brief-builder.js` + test | KEEP_IN_MAESTRO |
| GraphValidator / TaskGraphProposal / DAG utils | `runtime/planner/{graph-validator,task-graph-proposal,dag-utils}.js` | KEEP_IN_MAESTRO |
| PlanApprovalGate | `runtime/planner/plan-approval-gate.js` | KEEP_IN_MAESTRO |
| AIInterviewer / DynamicInterviewer / intent-* | `runtime/planner/{ai-interviewer,dynamic-interviewer,intent-*,question-*,batch-*}.js` | KEEP_IN_MAESTRO |
| SemanticRanker (context) | `runtime/context/semantic-ranker.js` | KEEP_IN_MAESTRO |
| ModelRouter | `runtime/planner/model-router.js` | KEEP_IN_MAESTRO (recommendation only) |
| PlanRevision* / PlanReview / PlanArtifact* | `runtime/planner/plan-*.js` | KEEP_IN_MAESTRO |
| TaskLifecycleMonitor / LaneExecutor / ReadinessEvaluator | `runtime/planner/*.js` | KEEP_IN_MAESTRO |

### MOVE_CONCEPT_TO_NEXUS (concept, re-impl in Go — no code port)

| Concept | Why |
|---|---|
| Task assignment to agents (LaneExecutor) | Nexus owns Agents + Resource Scheduler; Maestro provides the plan, Nexus schedules execution |
| AgentRole definitions | Nexus domain (Agent model) |

### REIMPLEMENT (in Nexus, Go, when the gate arrives)

| Concept | Note |
|---|---|
| Mission semantic lifecycle (Mission → Brief → Plan → Agents → Verification) | Gate 7, BETA, feature-flagged off by default |
| Advice/process compliance rendering | Gate 6 (Nexus bridge over Maestro contract) |

### DISCARD (belongs to neither, or superseded)

| Item | Reason |
|---|---|
| TUI taskgraph views (`runtime/tui/views/taskgraph-view.ts`) | Nexus is Web-first (§11); TUI phased out |
| TUI shell taskgraph model | same as above |
| Any code importing the WIP's bespoke CLI/daemon conventions | superseded by Nexus Web + REST/WS |

## Rules honored

- **No import/runtime dependency** of Maestro on Nexus (charter §4).
- **No forking** of Maestro methodology into Nexus (charter §5, §51): Nexus consumes
  a future machine-readable contract (Gate 6) rather than copying planner code.
- WIP branch is **not merged**; it is a salvage inventory for the Maestro owner to
  consolidate into its own `feat/nexus-contracts-v1` branch (§103).

## Next step

`DEV/NEXUS_V1_MAESTRO_INTEGRATION.md` defines the Nexus→Maestro bridge contract that
makes this salvage consumable without code porting.
