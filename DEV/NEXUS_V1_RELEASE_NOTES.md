# IAPro Nexus V1.0 Release Notes

**Release**: `v0.5.0-beta.4`  
**Commit**: `50bf05d` (synchronized with source HEAD)  
**Status**: Production-Ready Release Candidate

## Key Highlights
1. **Nexus Intelligence Layer**: Provider-neutral intent analysis, ambiguity evaluation (BLOCKING/IMPORTANT/LOW_IMPACT), context assembly, and deterministic prompt compilation.
2. **Structured WorkPlans & Prompt Compilation**: Hierarchical decomposition (`Mission -> Phase -> WorkPackage -> Task`), versioned immutable `PlanRevision`s, and `ExecutionSnapshot`s backed by SQLite.
3. **Visual Plan Builder**: Interactive Workspace OS interface for editing, reordering, and compiling prompts for structured work packages.
4. **Autonomous Mission Runner**: Durable state machine (`READY -> ALLOCATE -> COMPILE -> EXECUTE -> TEST -> REVIEW -> VERIFY`), bounded retries, and independent review verification loop.
5. **Resource Recommendation Service**: Multi-criteria explainable scoring (`BALANCED`, `PRESERVE_QUOTA`, `PREFER_PROVIDER`, `MANUAL`) based on real-time quotas and health.
6. **Hardened Security & Single-Writer Authority**: Unified terminal writer leases, loopback/private interface enforcement, session expiration/rotation, and Git branch protection.
7. **Web-First CLI Experience**: Default web launch (`nexus` / `ai`) with non-interactive subcommands (`plan`, `agents`, `projects`, `doctor`, `status`).
