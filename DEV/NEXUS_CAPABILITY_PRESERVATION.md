# Nexus Capability Preservation Audit

**Authority:** `docs/superpowers/specs/2026-09-01-nexus-canonical-product-contract.md`
**Functional implementation checkpoint:** `1e6404ac3dc487b78b1439a731a5ad47d16a1933`
**Branch:** `feat/nexus-canonical-review-rebuild`
**Audit date:** 2026-09-01

This audit is based on the current source tree and executable tests available in the local environment. It is not derived from historical status documents.

## Preservation verdict

No intentional baseline capability was removed in the canonical rebuild. New Workspace/Composer/Flow features are façades or presentation layers over the existing Project, Agent, Resource Scheduler, WorkPlan and Mission Runner subsystems.

| Capability | Current evidence | Verdict |
|---|---|---|
| Project create/register/manage | `internal/nexus/store`, `handlers_nexus.go`, `ProjectManagerSurface`, `AddProjectModal` | PRESERVED |
| Filesystem browse/inspect/scan/mkdir | `DirectoryBrowserModal`, `ProjectScanModal`, Nexus FS API | PRESERVED |
| Open project in OS file manager/editor/external terminal | `ProjectManagerSurface` + open-OS API | PRESERVED |
| Branch context/switcher | `BranchSwitcherModal` and project Git APIs | PRESERVED |
| Providers/profiles/accounts/auth/capabilities | control driver/profile APIs and Settings/Resource surfaces | PRESERVED |
| Usage/quota/reset/cache truthfulness | existing resource/usage models and recommendation pipeline; no synthetic token/cost data added | PRESERVED |
| Resource Scheduler + manual allocation | existing recommendation/allocation paths reused by Flow `AUTO`/manual restrictions | PRESERVED |
| Persistent Agents | Agent store + stable AgentID; Ask targets existing AgentID | PRESERVED + EXTENDED |
| Runtime generations/reconnect/recovery | existing Agent runtime-generation and recovery services | PRESERVED |
| Safe Apply/isolation/worktrees | existing mission execution/isolation code | PRESERVED |
| Terminal ANSI/input/resize/control | existing TerminalPane/protocol/SessionHost; Project Shell reuses it | PRESERVED + EXTENDED |
| Windows ConPTY/named pipes | existing platform-specific terminal backend/source | PRESERVED; NATIVE VALIDATION EXTERNAL |
| Intelligence CLI/API | existing intelligence service; now receives bounded/redacted context envelope | PRESERVED + HARDENED |
| WorkPlan revisions/diff/restore | `plans.go`, immutable PlanRevision APIs, PlanBuilder advanced compatibility view | PRESERVED |
| Clarification checkpoints | `clarifications` store + Composer/PlanBuilder flow | PRESERVED |
| Dependencies/parallel groups | WorkPackage model + canonical Flow DAG façade | PRESERVED + EXPOSED |
| Mission durability/fencing/retry | existing durable Mission Runner | PRESERVED + HARDENED |
| Verification/independent reviewer | existing Mission Runner review/verification path; no `reviewer-auto` added | PRESERVED |
| Take Control/Pause/Resume/Cancel | existing runner APIs surfaced in FlowRunSurface when supported | PRESERVED + EXPOSED |
| Schedules | existing schedules store/API/UI retained | PRESERVED |
| Maestro | independent adapter/catalog/gates retained; Nexus does not manufacture skills or semantic memory | PRESERVED |
| Updates | existing system-update APIs/UI retained | PRESERVED |
| CSRF/Origin/session secret boundaries | existing one-time bootstrap, HttpOnly session, CSRF and Origin checks retained | PRESERVED |
| Secret redaction | existing redactors plus bounded intelligence context envelope | PRESERVED + HARDENED |

## Canonical additions that do not replace the preserved Core

- Workspace V2 introduces stable `viewId` and `logicalKey` while preserving the existing surface tree.
- Tabs and Desktop are two presentations of the same logical Views.
- Project Shell reuses the supervised terminal runtime and is intentionally hidden from the normal AI-provider catalog.
- Composer replaces legacy product-mode language without deleting direct Agent work or the underlying WorkPlan machinery.
- Flow is a reversible façade over WorkPlan/WorkPackage/PlanRevision/MissionRun.
- ContextCapsule/WorkReceipt are additive evidence contracts attached to the existing durable Mission Runner.
- FlowRunSurface is a workspace façade over the persisted Mission Run, not a second runner.

## Known validation boundaries

- Full Go 1.25 compile/test/vet cannot run in this sandbox because the required toolchain and several modules are unavailable and outbound network/DNS is blocked.
- Native Windows and macOS execution are not proven by Linux source inspection or cross-platform source presence.
- Real Codex/Claude/Gemini/AGY/OpenCode provider E2E is intentionally separate from deterministic Core E2E.
- Release signing/distribution is not performed in this branch.

These are platform-validation gates, not evidence of a local functional regression.
