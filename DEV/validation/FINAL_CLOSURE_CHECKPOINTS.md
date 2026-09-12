# Nexus Final Closure Checkpoints

## Work Package B — Native Skills foundation

- `work_package`: B
- `status`: GREEN (foundation slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/skills/{types,catalog,sources}.go`, `internal/nexus/{skill_catalog,skill_contract}.go`, WorkPlan/Flow/runner transport, and focused Nexus tests
- `tests_run`: `go test ./internal/nexus/skills -count=1`
- `test_results`: PASS after RED → GREEN; initial RED was an expected compile failure before implementation.
- `decisions`: canonical source-agnostic `Skill`, `Source`, bounded SKILL.md parser, deterministic version/source deduplication, optional source degradation.
- `evidence`: builtin resolution and package freeze without Maestro, deterministic
  version/deduplication, bounded markdown discovery, optional-source unavailability,
  and canonical SkillIDs transport through Flow/runner/capsule.
- `known_risks`: Maestro and Composer still expose some legacy compatibility contracts; project/user source wiring and full resolution provenance remain pending.
- `remaining_work`: encapsulate/remove remaining legacy Maestro-named consumer contracts, add store-level skill resolution evidence, and complete project/user source provenance.
- `unlocked_dependencies`: generic catalog is now available to Composer and routing work.

## Work Package C/D — Project Intelligence and explicit intent routing

- `work_package`: C/D
- `status`: GREEN (bounded integration slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/project_intelligence_context.go`, `internal/nexus/intent_router.go`, Composer/intelligence provider integration, CLI goal routing.
- `tests_run`: `go test ./internal/nexus/... -count=1`; focused intent and grounding tests.
- `test_results`: PASS.
- `decisions`: prefer the persisted identity-bound snapshot; bounded static discovery is the safe fallback; provider ambiguity analysis receives the same bounded envelope when supported; `DIRECT`, `CLARIFY`, and `PLAN` are deterministic and explainable.
- `evidence`: atomic test routes DIRECT; repository facts prevent ritual clarification; complete multi-step goals route PLAN; empty/material objective routes CLARIFY; grounded facts include basis/confidence/source.
- `known_risks`: persisted Project Intelligence may be absent and is synchronously discovered; external intelligence/provider and Composer E2E remain unverified.
- `remaining_work`: persist intent decisions as first-class mission evidence and expand provider-backed end-to-end tests.
- `unlocked_dependencies`: routing can now feed WorkPlan creation and runtime resolution.

## Work Package E/G — Runtime affinity and routing decision

- `work_package`: E/G
- `status`: GREEN (deterministic resolution and mission persistence slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/runtime_routing.go`, runner/store WorkPackage contracts, `internal/nexus/mission_executor.go`.
- `tests_run`: `go test ./internal/nexus/... -count=1`; focused RuntimeRouting tests.
- `test_results`: PASS.
- `decisions`: AUTO/PREFER/PIN are explicit; desired provider/profile are copied into durable package state before actual allocation; fallback does not overwrite desired affinity; PIN returns `BLOCKED_RESOURCE`; task class chooses the cheapest capable model candidate.
- `evidence`: PREFER fallback preserves desired values; PIN unavailable fails closed; coding selects cheap capable model; architecture raises reasoning floor; package runs persist `routing_decision` JSON.
- `known_risks`: current scheduler still owns the final account allocation; model inventory is a contract-level candidate set and live provider/model evidence is unavailable.
- `remaining_work`: add store-level query/report projection for routing decisions and append mission verification entries to the canonical evidence ledger.
- `unlocked_dependencies`: explainability can be surfaced without changing Agent identity.

## Work Package L — Canonical validation evidence integration

- `work_package`: L
- `status`: GREEN (integration slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/validation_evidence.go`, `internal/nexus/validation_evidence_test.go`, `internal/nexus/runner/{executor,runner}.go`.
- `tests_run`: `go test ./internal/nexus/... -count=1`; `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make quality-full`.
- `test_results`: PASS. Package and global verification results now append to the existing `ValidationEvidenceStream`, bind identity/environment/provider/profile/model where available, and verify the hash chain before promotion.
- `decisions`: recorder is an optional runner capability to preserve existing test executors; the real Nexus executor fails closed with `VALIDATION_EVIDENCE_UNAVAILABLE`; non-Git or missing-SHA observations remain `OBSERVED/NOT_VERIFIED`.
- `evidence`: unit coverage for identity binding, stable idempotency ID, append-only store chain, CAS and tamper detection; full race and security gates green.
- `known_risks`: no completed production Mission was run in this campaign, so no durable stream instance ID or end-to-end mission evidence can be claimed.
- `remaining_work`: execute an authenticated Mission that emits a real stream ID and project/provider/model evidence; add report projections and broader global DoD coverage.
- `unlocked_dependencies`: final reports can consume the canonical ledger once a real Mission is executed.

## Work Package P — Final verification and release verdict

- `work_package`: P
- `status`: NO-GO (evidence blockers remain)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `web/src/features/work/ProjectIntelligenceInspector.tsx`, frontend verification reports, this checkpoint and handoff documentation.
- `tests_run`: `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make build`; `make build-desktop`; `make web-verify`; `git diff --check`; `./nexus doctor --json`; redacted real-provider smoke attempts.
- `test_results`: Go/frontend/security/build/race/vet gates PASS. Linux CLI and desktop builds PASS. `make web-verify` PASS with the current frontend report. AGY smoke PASS. Codex smoke reached the authenticated CLI but was rejected by provider usage limit; the other Codex profile has a local rules symlink loop.
- `decisions`: no PASS is assigned to Windows/macOS, overnight, OpenCode, or authenticated Mission E2E without direct evidence. No commit/push was created.
- `evidence`: `DEV/validation/FRONTEND_LATEST.md`, command outputs from the final gate runs, `./nexus doctor --json`, and the test suites above.
- `known_risks`: legacy Maestro-named compatibility contracts remain in Nexus types/Flow payloads; live account/model routing and durable explainability projection are partial; no real Mission stream ID was produced.
- `remaining_work`: remove/encapsulate remaining Maestro skill-type leakage; execute authenticated Mission scenarios 1–15; prove provider failover/PIN/model escalation/handoff/restart; native Windows/macOS and overnight runs.
- `unlocked_dependencies`: none for GO; campaign is correctly stopped at NO-GO until external/runtime evidence exists.

## Work Package B/E/G — native consumer decoupling and decision projection continuation

- `work_package`: B/E/G continuation
- `status`: GREEN (local implementation slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: generic Agent prompt path and builtin Skills, generic intelligence guidance, prompt variants, legacy gate/Skill boundary, mission routing projection, Codex identity continuity, Flow canonical `skill_ids` transport.
- `tests_run`: `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make build`; `make build-desktop`; `make web-verify`; `git diff --check`.
- `test_results`: PASS. Focused RED→GREEN tests also pass for builtin/project-independent Skills without Maestro, generic guidance, legacy gate separation, persisted routing selection, and Codex identity history.
- `decisions`: Agent execution now resolves through the source-agnostic SkillCatalog; `MaestroSkillDesc` is no longer accepted by prompt variant compilation; `ExecutionGuidance` is canonical with a legacy Maestro alias; `MaestroGates` are not interpreted as Skills; mission allocation persists desired affinity and actual selected provider/profile/model without mutating preference.
- `evidence`: backend full and race suites, frontend `make web-verify` report `DEV/validation/FRONTEND_LATEST.md`, Linux CLI/Desktop builds, and clean `git diff --check`.
- `known_risks`: live model inventory/escalation is still contract-level in the Mission executor; Maestro lifecycle gate advice remains compatibility-oriented; no authenticated Mission emitted a real evidence stream instance in this campaign.
- `remaining_work`: authenticated provider Mission scenarios, account/provider/model failover and handoff proof, dedicated routing report projection, native Windows/macOS execution, overnight acceptance, and external evidence reconciliation.
- `unlocked_dependencies`: local code-quality gates are green; external/runtime evidence remains the release blocker.

## Work Package C/E/I — intent and explainability projection continuation

- `work_package`: C/E/I continuation
- `status`: GREEN (local contract slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: intent decision metadata in existing WorkPlan structured facts, typed routing report projection and `/api/v1/runs/{id}/routing`, Web API client/types, Maestro lifecycle metadata.
- `tests_run`: focused Nexus/Web tests, `go test ./internal/nexus ./internal/control/web -count=1`, frontend routing API test and typecheck.
- `test_results`: PASS. Corrupt persisted routing JSON fails closed; valid decisions project selected provider/profile/model and reason; intent metadata preserves existing facts; degraded Maestro preserves lifecycle without fabricating advice.
- `decisions`: explainability is read from persisted decisions and never recomputed from live quota/health; the routing report reuses RunApplicationService; lifecycle is additive and defaults to `TASK`.
- `evidence`: `internal/nexus/routing_report_test.go`, `internal/nexus/intent_router_test.go`, `internal/nexus/maestro_test.go`, `web/src/nexus/api.test.ts`.
- `known_risks`: full task-aware live model escalation and lifecycle matrix remain incomplete; intent evidence is stored in WorkPlan metadata, but not yet projected as a separate ledger entry.
- `remaining_work`: complete model/escalation integration, append intent/closure evidence to the canonical ledger, and obtain authenticated Mission/native/overnight proof.
- `unlocked_dependencies`: consumers can query durable routing explanations without provider access.
