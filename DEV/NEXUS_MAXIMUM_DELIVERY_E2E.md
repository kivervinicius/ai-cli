# IAPro Nexus — E2E Contract Report

## Required closed-loop model

`UI -> HTTP/WS -> Service -> SQLite -> Runtime/Provider -> event/output -> HTTP/WS -> UI`

## Direct Work

Implemented flow:

1. Project Work surface opens Direct Session launcher.
2. User explicitly selects provider/profile; no Mission is created.
3. Frontend creates Persistent Agent.
4. `/api/v1/resources/select` persists MANUAL resource allocation.
5. `/api/v1/agents/:id/start` resolves persisted provider/profile.
6. RuntimeSession is created with `agent_id`.
7. Agent terminal connects by `AgentID` and resolves current RuntimeGeneration.
8. Initial prompt is sent only after CONTROL lease.

Real-provider validation harness: `scripts/nexus-e2e-local.go`.

The harness requires a running local Nexus and an already-authenticated provider. It verifies bootstrap/session, Project, resource, Agent, runtime identity, WebSocket CONTROL lease, prompt send, provider output, durable RuntimeGeneration linkage and stop/cleanup.

## Reconnect / Safe Apply

- Agent terminal URL is Agent-scoped.
- RuntimeGeneration change causes bounded client reconnect to the current generation.
- Broker and SessionHost CONTROL lease are synchronized over the attached IPC connection.
- New generations have truthful continuity semantics.

## Mission

Implemented contracts include:

- approved/frozen plan revision;
- PromptVersion;
- ResourceAssignment;
- Persistent Agent allocation;
- worktree execution isolation;
- execution evidence (`RuntimeID`, provider output excerpt, changed paths);
- testing;
- independent review when capability permits;
- final verification;
- remediation/retry from the correct failed stage;
- DAG dependency release and parallel groups;
- terminal success only as `COMPLETED_VERIFIED` after verification evidence.

## Restart

- RunRepository is durable.
- lease has owner, TTL and fencing token.
- expired lease can be reclaimed after restart.
- active lease cannot be reacquired concurrently, even by the same MissionRunner owner.
- run/snapshot state is restored rather than reconstructed from UI state.

## Take Control

- Take Control refuses a package without a real assigned Agent.
- runner pauses package ownership.
- manual checkpoint records before/after workspace fingerprints and changed paths.
- Return to Mission resumes at testing when valid manual work changed the workspace, avoiding duplicate implementation.

## Scheduling

- Start now, time-based and after-run scheduling contracts exist.
- due schedule uses an atomic claim before starting a run to prevent duplicate execution.
- frontend sends the same explicit AutonomyContract snapshot for immediate and scheduled runs.

## Environment limitation

The real-provider harness was **not executed inside this sandbox**, because no authenticated coding provider exists here. This is an external release gate, not represented as PASS.
