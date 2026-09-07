# Composer → Flow contract

This campaign evolves the Nexus Composer only. The Flow canvas, WorkPlan,
Mission Runner, Scheduler and runtime remain owned by their existing modules
and are read-only integration targets.

The Composer owns the durable discovery revision, Motivation Map, provenance,
prompt artifacts, prompt variants, destination receipts and the explainable
FlowSuitability assessment. Maestro remains authoritative for method, process,
canonical skills, gates and risk advice. A missing Maestro is surfaced as
`MAESTRO_DEGRADED`; the Composer does not synthesize skills or advice.

`PromptArtifact → MaterializePromptArtifactAsFlow` is the existing handoff.
The handoff carries the artifact identity, revision, content hash, structured
brief, Motivation Map, selected skills and context in WorkPlan facts. It creates
a draft WorkPlan only; it does not start an Agent, MissionRun or runtime.

The visual Flow is a projection, not a second Composer plan. Any future Flow
editor mutation must return through the canonical WorkPlan contract. Layout is
not execution semantics.

Legacy Composer sessions remain readable: absent revision/provenance/variant
fields are represented by compatibility defaults and are never destructively
migrated.
