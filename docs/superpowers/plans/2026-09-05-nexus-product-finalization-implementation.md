# Nexus Product Finalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** close the verified P0/P1 safety and release gaps without rewriting capabilities that already pass their focused tests.

**Architecture:** Keep existing Composer, Flow, Mission Runner and Workspace models. Add fail-closed guards at execution boundaries, make preflight and schedule claims authoritative, restore the missing shared token module, and record unverified cross-platform/E2E capabilities rather than simulating them.

**Tech Stack:** Go 1.25, SQLite, React 19, TypeScript, Sass modules, Vitest, Bun lockfile.

**Spec:** `docs/superpowers/specs/2026-09-05-nexus-product-finalization-audit.md`

## Global Constraints

- Autonomous writers must use a dedicated worktree and fail closed if isolation cannot be established.
- WorkPlan/FlowRevision remains the canonical execution graph.
- No provider-name whitelist may decide capabilities.
- New component styles use SCSS Modules and semantic CSS custom properties.
- No merge, push, tag, release, or publish is performed by this campaign.
- Every behavior change gets a failing regression test before implementation.

### Task 1: Restore the frontend token contract
- [x] Write the minimal token alias module required by existing SCSS imports.
- [x] Run `npm run build` and verify the previous missing-stylesheet failure is gone.
- [x] Run `npm run quality:full` and commit `fix(ui): restore shared sass token contract`.

### Task 2: Enforce autonomous worktree isolation
- [x] Add a failing test proving an autonomous mission cannot resolve to the canonical project checkout.
- [x] Make autonomous execution require `Isolation == "worktree"` and return an actionable error otherwise.
- [x] Run focused executor tests and commit `fix(flow): fail closed without autonomous worktree`.

### Task 3: Make schedule claiming atomic
- [x] Add a failing concurrent-claim test with two workers and one pending schedule.
- [x] Add an atomic claim/update guarded by pending status before creating a run.
- [x] Verify only one run is created and commit `fix(flow): atomically claim scheduled runs`.

### Task 4: Make preflight and Maestro enforcement real
- [x] Add failing tests for invalid provider commands, unvalidated Maestro skills, and approved runs bypassing preflight.
- [x] Remove hardcoded PASS results; validate step resources, isolation, autonomy and available project tooling.
- [x] Require successful preflight/frozen gates in approved run path and commit `fix(flow): enforce preflight and maestro gates`.

### Task 5: Preserve PromptArtifact lineage and contextual decomposition
- [x] Add failing tests proving `ArtifactID` and version affect the generated Flow lineage.
- [x] Reject or explicitly report unsupported generic verification commands instead of injecting them blindly.
- [x] Commit `fix(flow): preserve artifact lineage during decomposition`.

### Task 6: Produce the final evidence report
- [x] Re-run all available gates and record exact outputs.
- [x] Classify capabilities 1–70 with L1–L8 evidence and mark untested platform/E2E claims accordingly.
- [x] Record commits, remaining risks, G0–G10 verdicts and reproduction commands; commit `docs(audit): record nexus product finalization evidence`.
