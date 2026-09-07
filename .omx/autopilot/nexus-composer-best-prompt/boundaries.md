# Composer Autopilot boundaries

- In scope: Composer domain, PromptArtifact/revision contracts, provenance,
  context snapshots, skill catalog adapters, prompt compilation, destination
  receipts, Flow handoff adapter, Composer UI and tests.
- Out of scope: Flow canvas internals, WorkPlan, MissionRunner, Scheduler and
  Maestro core semantics.
- Flow files are read-only for this campaign. Only an agreed handoff contract or
  fixture may be changed if required by the Composer integration.
- Preserve all existing worktree changes. No reset, destructive checkout,
  destructive removal, commit or push.
- Maestro remains authority for method, process, risk, gates and canonical skills.
