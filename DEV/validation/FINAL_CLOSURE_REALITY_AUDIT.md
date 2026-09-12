# Nexus Final Closure — Reality Audit

Date: 2026-09-11
Base SHA: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
Branch: `feat/nexus-maximum-delivery`
Worktree at audit start: clean

This is the evidence-backed characterization for the final consolidation
campaign. It is not a release claim. Local Linux checks are separated from
provider-authenticated and native-platform evidence. The audit was updated
after the implementation slices; historical “ADD/EXTEND” actions below are
the starting characterization, while the resumable result is in
`FINAL_CLOSURE_CHECKPOINTS.md`.

## Capability matrix

| Capability | Existing | Evidence | Action |
| --- | --- | --- | --- |
| Project Intelligence | `contextsnapshot.Discover`, immutable `ProjectContextSnapshot`, identity digests, bounded static scan, SQLite persistence and Web read model | `internal/nexus/contextsnapshot/*_test.go`, `project_intelligence.go`, `store/project_intelligence.go`; baseline `go test ./internal/nexus/...` PASS | EXTEND: enrich inventory and expose the persisted snapshot to Composer without creating a second scanner |
| Native Skills | Generic `internal/nexus/skills` catalog/sources now exist; legacy Maestro catalog remains for compatibility | `internal/nexus/skills/*`, `skill_catalog.go`, focused tests | EXTEND: migrate remaining legacy consumer/storage contracts without breaking compatibility |
| Intent Router | Explicit deterministic `DIRECT | CLARIFY | PLAN` decision now sits above existing requirements | `intent_router.go`, focused tests, `internal/app/app.go` | EXTEND: persist decision as first-class mission evidence |
| Composer grounding | Composer receives a bounded identity-bound Project Intelligence envelope for intent/ambiguity analysis | `project_intelligence_context.go`, `composer.go`, focused tests | EXTEND: provider-backed E2E and relevance selection |
| Agent matching | Provider-independent `MatchAgents`, hard gates, score and deterministic ID tie-break | `agent_matching.go`, `agent_matching_test.go` | REUSE; add routing integration tests only |
| Runtime affinity | Canonical `AUTO/PREFER/PIN` desired-vs-actual resolver exists beside the existing scheduler | `runtime_routing.go`, focused tests | EXTEND: wire all live account/model inventory and engine policy |
| Resource routing | `ProviderAccount`, quota confidence, health/cooldown/capability hard gates and deterministic recommendation | `resource_recommendation.go`, `resource_discovery.go`, quota tests | REUSE; consume affinity policy |
| Model routing | Task-aware candidate selection is deterministic and cost-first with reasoning floor | `runtime_routing.go`, focused tests | EXTEND: live health/quota/escalation integration |
| Routing Decision | Typed decision is persisted as bounded `routing_decision` JSON on package state | `runtime_routing.go`, runner/store contracts, mission executor | EXTEND: query/report projection and full runtime provenance |
| Handoff | `ContextCapsule`, `WorkReceipt`, runtime generations, same-provider honest `NATIVE_RESUME_UNVERIFIED`, cross-provider context handoff | `runner/evidence.go`, `continuity.go`, handoff docs/tests | REUSE; add routing provenance to the existing capsule/receipt only where needed |
| Maestro lifecycle | Advice/status/catalog and optional compiler guidance exist; lifecycle advice is not typed as PLAN/TASK/RECOVERY/VERIFY | `maestro.go`, `handlers_maestro.go`, intelligence context | EXTEND: lifecycle field/adapter, preserve Nexus-owned resource choice |
| Attention | Durable MissionRunner intervention, idempotent resolution, global runner attention tests and Web runtime attention surfaces | `runner/intervention_test.go`, `runner/attention_e2e_test.go`, Web attention files | REUSE; verify current integration and do not create a second state machine |
| Evidence | Existing append-only ledger is now consumed by package/global runner verification and chain-verified before completion | `store/validation_evidence.go`, `validation_evidence.go`, runner optional recorder, tests | EXTEND: produce a real authenticated Mission stream instance and report projections |

## Confirmed invariants

- `AgentSpec` is already separate from provider/model/account.
- `WorkPlan` is the semantic source; `Flow` is its projection.
- `ContextCapsule` and `WorkReceipt` are bounded evidence contracts, not raw chat history.
- `BLOCKED_NEEDS_USER` and `FAILED_NO_PROGRESS` are distinct runner states.
- Local baseline is green for the focused Core/Intelligence/Runner slice; this
  does not prove live providers, overnight autonomy or native Windows/macOS.

## Current blockers to GO

1. Legacy Maestro-named skill/storage contracts still exist in compatibility
   payloads and the Maestro catalog adapter; canonical Agent execution and
   prompt compilation are source-agnostic, but complete storage migration is
   not claimed.
2. Intent and routing decisions are explainable in persisted package JSON, but
   intent and routing are not yet exposed through a dedicated report/query
   projection.
3. The effective Mission path still combines `RecommendResources` with the
   existing scheduler; the unified live allocation contract is not fully
   proven.
4. Model/engine/account provenance and escalation remain contract-level for
   some providers; live health/quota integration is incomplete.
5. No authenticated Mission produced a durable evidence stream ID in this
   campaign. Real provider, overnight, and same-SHA native-platform proof
   remains unavailable or partial and stays `UNVERIFIED`.

## Baseline commands

| Command | Result |
| --- | --- |
| `git fetch --all --prune` + checkout/pull/status/rev-parse/log | PASS; base SHA recorded above |
| `go test ./internal/nexus/... -count=1` | PASS |
| `go vet ./...` | PASS |
| `npm --prefix web run typecheck` | PASS |
| `git diff --check` | PASS |
