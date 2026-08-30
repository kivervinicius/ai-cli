# IAPro Nexus — Extreme Local Production Validation Prompt

Use this prompt with an autonomous coding/QA agent from the root of the final repository. Do not modify production code merely to make a test pass unless a reproducible defect is found. Preserve all evidence under `DEV/validation/`.

## Mission

Act as an independent senior Release Engineer + QA + Security reviewer. Determine whether this exact commit is fit for production. Never infer PASS from file existence, mocks, reports or previous runs. Re-run every gate.

## 1. Identity and cleanliness

```bash
git status --porcelain
git branch --show-current
git rev-parse HEAD
cat VERSION
git diff --check
```

Fail release if the tree is dirty before validation or if generated Web assets do not come from the same commit.

## 2. Go 1.25 mandatory gates

```bash
go version
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/nexus
```

Require Go >= 1.25.0. Record exact outputs and exit codes.

## 3. Frontend mandatory gates

From `web/`:

```bash
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run test
bun run build
```

Then prove `web/dist/bundle.js` and `internal/control/web/dist/bundle.js` hashes match, likewise CSS.

## 4. Direct Work real provider E2E

Start Nexus normally. Use a provider/profile already authenticated on the machine and execute:

```bash
NEXUS_BOOTSTRAP_URL='PASTE_ONE_TIME_BOOTSTRAP_URL' \
NEXUS_E2E_PROJECT_PATH='/absolute/path/to/a/safe/test/git/repo' \
NEXUS_E2E_PROVIDER='claude' \
NEXUS_E2E_PROFILE='default' \
go run ./scripts/nexus-e2e-local.go
```

Repeat for each installed supported provider where practical: Codex, Claude, Gemini, AGY, OpenCode, Cursor.

PASS requires: authenticated resource -> Persistent Agent -> `runtime.agent_id` -> Agent WebSocket CONTROL -> prompt -> provider output -> durable RuntimeGeneration -> verified stop.

## 5. Safe Apply / reconnect

With an Agent terminal visibly running:

1. change a restart-required supported config;
2. preview impact;
3. apply;
4. verify old generation terminates;
5. verify new RuntimeGeneration has the same AgentID;
6. verify terminal reconnects automatically;
7. verify no new process is labelled `LIVE_SAME_RUNTIME`;
8. verify only one CONTROL writer can type.

## 6. Context handoff

Use two actually installed compatible providers. Verify provider A snapshot/context is compiled into provider B **NEW SESSION**. Never accept cross-provider operation labelled native resume.

## 7. Mission full E2E

Create a safe sample product with at least three packages:

- package A foundation;
- package B and C depend on A and share a parallel group.

Approve explicit AutonomyContract with destructive/network/secret/deploy/paid permissions disabled unless the test specifically needs them.

Require evidence of:

- frozen PlanRevision;
- PromptVersion hashes;
- scheduler allocation respecting capabilities and locks;
- distinct Agent worktrees;
- real provider execution;
- changed files;
- tests;
- independent reviewer;
- VERIFYING stage;
- `COMPLETED_VERIFIED` only after final verification.

## 8. Restart durability

During a package execution:

1. terminate Nexus process without deleting data;
2. restart Nexus;
3. wait for lease expiration/reclaim;
4. prove the existing MissionRun is resumed from durable snapshot;
5. prove no package runs twice concurrently;
6. verify fencing token rejects stale worker writes.

## 9. Failure/remediation

Create a deterministic failing test in the sample mission. Require:

`TESTING -> failure evidence -> DIAGNOSING/REMEDIATING -> new prompt/execution -> TESTING -> REVIEWING -> VERIFYING`.

Fail if state merely changes without a provider execution/artifact difference.

## 10. Take Control

During Mission execution:

1. Take Control;
2. verify package pauses and exposes the actual assigned Agent;
3. edit code manually in its worktree;
4. Return to Mission;
5. verify before/after fingerprints + changed paths are stored;
6. verify Runner resumes at testing and does not duplicate the implementation prompt.

## 11. Scheduling

Verify Start Now, absolute date/time and After another Mission. Simulate two scheduler workers against the same due schedule. Exactly one may claim/start it.

## 12. Security

Actively test:

- Origin mismatch;
- host/port mismatch;
- CSRF absent/bad/rotated;
- session logout and rotation;
- WebSocket without cookie;
- directory traversal and symlink escape;
- Agent/Project IDOR attempts;
- branch values beginning with `-`;
- raw secrets in HTTP responses/logs/config;
- destructive Git with policy disabled;
- push/deploy/network/secret/paid CLI guards.

Explicitly report that PATH command guards are policy enforcement, not hostile-code sandboxing. Do not mark VM/container-grade isolation PASS unless such a mechanism is actually enabled and tested.

## 13. Platform matrix

Run the same final commit on:

- Ubuntu latest;
- macOS latest;
- Windows latest.

Required OS-specific evidence:

- Linux/macOS PTY + Unix socket + resize + lifecycle;
- Windows ConPTY + Named Pipe + resize + lifecycle;
- browser launch;
- install/uninstall smoke;
- provider detection/profile/resource discovery.

## 14. CI / release

Push only to a review branch. Require all final CI matrix jobs green. Run GoReleaser snapshot and validate archives/checksums. Confirm build metadata contains the final SHA and version.

## 15. Final report

Create `DEV/validation/FINAL_PRODUCTION_VALIDATION.md` with exact commands, exit codes, OS, versions, provider versions and evidence links.

Final verdict must be exactly one of:

- `GO`
- `CONDITIONAL_GO`
- `NO_GO`

Use `GO` only if every mandatory production gate above has real evidence on the final commit.
