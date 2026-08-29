# IAPro NEXUS — WORKSPACE OS FINALIZATION AUTOPILOT

## ROLE

Act as the final senior engineering owner, independent QA engineer, accessibility reviewer and release hardening engineer for **IAPro Nexus — Powered by Maestro**.

You are not starting a redesign and you are not allowed to reset the repository to an older baseline. The Workspace OS direction is already approved. Your job is to **inspect, execute, test, diagnose, correct and finish the current branch until the implemented behavior matches the approved contract, or produce a technically proven NO-GO**.

Operate in AUTOPILOT. Do not repeatedly ask the user to approve ordinary implementation, test, refactor, review, worktree, local commit, retry or remediation actions that are already inside this prompt. Ask only for genuinely non-resolvable product decisions, destructive external actions, new secrets/credentials, paid external services, production deployment, protected-branch push/merge, or requirements that would materially contradict the approved architecture.

---

# 1. STARTING POINT — PRESERVE IT

Open the repository from this handoff ZIP and inspect Git before changing anything.

Expected handoff branch is similar to:

```text
feat/nexus-workspace-os-handoff
```

Do NOT assume the exact HEAD; determine it with Git.

Read first:

```text
DEV/NEXUS_WORKSPACE_OS_HANDOFF.md
docs/superpowers/specs/2026-08-29-nexus-workspace-os-design.md
docs/superpowers/plans/2026-08-29-nexus-workspace-os.md
```

If present, also inspect:

```text
docs/design/reference-workspace-os-prototype.html
```

That HTML is a visual/product reference, not an instruction to revert to a static mockup.

Before modifications record:

```bash
git status --short
git branch --show-current
git rev-parse HEAD
git log --oneline --decorate -20
```

Preserve all existing post-baseline backend work, including current Project/Agent/runtime, Resource Scheduler, Maestro, Missions, terminal broker and Agent configuration evolutions. Never reset to `82470ff`, `f9cd679`, `main`, or another earlier commit merely because it is clean.

---

# 2. MANDATORY ENGINEERING METHOD

Use installed **Superpowers** skills and the repository's **Maestro** capabilities rather than simulating them from memory.

At minimum discover/use, when installed/applicable:

```text
using-superpowers
systematic-debugging
test-driven-development
verification-before-completion
receiving-code-review
requesting-code-review
finishing-a-development-branch
```

Use an isolated worktree if the current harness supports it and the branch is not already isolated.

The design decision is already approved, so do NOT restart brainstorming around whether the product should be a dashboard, free-floating-window desktop or Workspace OS. The approved answer is the Workspace OS described below.

For every defect:

```text
reproduce -> identify root cause -> write/adjust failing test where testable -> verify RED -> implement minimal correct fix -> verify GREEN -> regression suite
```

Do not claim completion from code inspection alone.

---

# 3. PRODUCT CONTRACT — DO NOT REGRESS

IAPro Nexus is a **Project-first AI development workspace**, not a generic admin dashboard and not a full IDE.

The approved shell is:

```text
GLOBAL NEXUS SHELL
        ↓
PROJECT WORKSPACE
        ↓
DOCKABLE SURFACES
        ↓
TERMINALS / WORK / PLAN / AGENTS / MAESTRO / RESOURCES / SESSIONS
```

Core product direction that must remain intact:

```text
User Goal
   ↓
Nexus Intelligence (future/current progressive capability)
   ↓
Maestro — process / skills / risk / verification
   ↓
WorkPlan / Mission
   ↓
Resource Scheduler
   ↓
Persistent Agents
   ↓
Provider runtimes / terminals
```

The current finalization scope is primarily **Workspace OS correctness + integration**, while preserving architectural seams for Intelligence, WorkPlan, Missions and autonomous execution.

Do NOT add Monaco, a full source-code editor, a VS Code clone, a generic Git IDE, or unrelated feature creep.

---

# 4. APPROVED WORKSPACE OS UX

The Web UI must feel like a development workspace / lightweight operating environment, while remaining accessible and understandable.

Required behaviors:

### Global / Project shell

- Project-first navigation.
- Persistent Project Rail on desktop.
- Responsive Project selector/drawer on constrained widths.
- Project context visible in the shell.
- Branch/repository context where available.
- Global access to Resources, Maestro, Sessions and Settings.
- Project-level access to Overview, Work, Plan/Missions, Agents, Maestro, Usage/Events where implemented.
- `Ctrl/Cmd+K` command palette.
- Bottom taskbar/dock for opened workspace surfaces.

### Workspace

- Tabs.
- Multiple tab stacks.
- Horizontal and vertical splits.
- Drag/move surfaces between tab stacks.
- Resize split panes.
- Keyboard-accessible split resize.
- Maximize/restore a stack.
- Close visual surface without accidentally killing an Agent/runtime.
- Popout/detach surface to another browser window where browser permits.
- Restore persisted layout for a Project.
- Project A layout must not leak into Project B.
- Returning from mobile/compact layout must not destroy the saved desktop layout.

### Terminal

- Terminal surface is conceptually attached to stable `AgentID`, not a disposable UI tab identity.
- Switching ordinary Workspace tabs must not stop the Agent.
- Moving terminal between stacks must not create a new provider process.
- Resize must refit xterm when the pane size changes, not only on browser resize.
- CONTROL vs VIEW_ONLY must remain truthful.
- VIEW_ONLY must not write terminal input.
- Backend AgentTerminalBroker/runtime-generation continuity must be tested with real runtime transitions where supported.

### Surfaces

At minimum verify real integration for:

```text
Project Overview
Work
Agents
Agent Configuration
Agent Terminal
Resources
Maestro
Plan / Missions
Sessions / Lineage
Settings
```

Legacy Runtime/Provider/Event controls may remain available as advanced surfaces; do not make them the primary experience again.

---

# 5. DESIGN SYSTEM AND UI ARCHITECTURE

Audit the frontend organization rather than collapsing it back into monolithic files.

Expected separation includes concepts similar to:

```text
web/src/app/
web/src/design-system/
web/src/features/
web/src/workspace/
web/src/nexus/
web/src/shared/ or existing equivalents
```

Reusable primitives and patterns should be used instead of repeated page-specific Tailwind strings wherever practical.

Required theme capabilities:

```text
System
Dark
Light
High Contrast
```

Required personalization:

```text
accent: Purple / Blue / Cyan / Neutral
UI density: Compact / Comfortable
reduced motion preference
```

Theme choices must persist.

Do not make status distinguishable only by color.

## Optional library hardening

The handoff intentionally does **not** inject new network-only dependencies into an environment that could not install them.

If the finalization machine has reliable package-network access, evaluate — do not blindly install — whether replacing internal primitives with these improves maintainability without changing UX contracts:

```text
FlexLayout React — docking/split/tab infrastructure
Radix UI Primitives — dialogs/popovers/selects/focus/a11y
React Joyride — guided tour
```

Migration is OPTIONAL. Keep the current internal implementation if it is correct, accessible, tested and simpler. Do not destabilize a working Workspace OS merely to satisfy a library name.

---

# 6. PRODUCT TOUR / EDUCATION

Verify a first-use tour and replay mechanism.

The tour should explain, in concise contextual steps:

1. Projects.
2. Project Workspace.
3. Persistent Agents.
4. Terminals and CONTROL/VIEW_ONLY concept.
5. Resources/providers/quota.
6. Maestro.
7. Work entry point.
8. Plan/Missions.
9. Command Palette.
10. Themes/accessibility/personalization.

The tour must:

- not trap users permanently;
- be dismissible;
- be replayable;
- restore focus correctly;
- not point to targets that do not exist in the current viewport;
- behave reasonably on mobile.

Tooltips/help copy should explain Nexus-specific technical concepts without exposing implementation jargon unnecessarily.

---

# 7. REAL BROWSER VISUAL QA — MANDATORY

The handoff environment could compile/test the Web app but could not obtain trustworthy browser rendering because organization policy blocked localhost in its Chromium sandbox.

You MUST perform rendered-browser QA on the finalization machine.

Use Playwright if already available or install/use it if the environment permits. Otherwise use the browser automation tooling available in the harness.

Validate at least:

```text
1600x1000 desktop
1366x768 laptop
1024x768 tablet
390x844 mobile
```

Inspect both the real application and, when useful, the clearly synthetic:

```text
?demo=1
```

Demo mode must remain obviously synthetic and must never perform real Agent/provider mutations.

For each viewport capture screenshots of representative states and inspect them, not merely assert that the page loaded.

Verify visually:

- no clipped navigation;
- no hidden critical actions;
- no accidental horizontal page scroll;
- tab labels remain usable;
- terminal gets usable space;
- taskbar does not cover content;
- panels can resize without layout collapse;
- dialogs/palette fit viewport;
- Project Rail behavior is sensible;
- text contrast works in every theme;
- light theme is genuinely designed, not an inverted afterthought;
- High Contrast visibly increases contrast/focus;
- typography/spacing feel cohesive;
- UI is recognizably an evolution of the supplied prototype, not a generic admin dashboard.

Create:

```text
DEV/NEXUS_WORKSPACE_OS_VISUAL_QA.md
```

Include screenshot paths/artifact references and findings/fixes.

---

# 8. ACCESSIBILITY QA — MANDATORY

Perform automated and manual accessibility validation.

If axe-core/Playwright axe tooling is available, run it. Do not use absence of axe as an excuse to skip manual checks.

Verify at minimum:

- keyboard-only navigation;
- visible focus;
- no keyboard trap;
- `Ctrl/Cmd+K` palette open/search/arrows/Enter/Escape;
- tour focus and Escape behavior;
- dialogs restore focus;
- tabs expose appropriate semantics;
- splitters expose semantics and keyboard resize;
- buttons have an accessible name;
- icon-only buttons have labels;
- status isn't color-only;
- reduced motion is honored;
- 200% zoom remains usable;
- High Contrast mode remains readable;
- responsive touch targets are reasonable.

Create:

```text
DEV/NEXUS_WORKSPACE_OS_ACCESSIBILITY.md
```

Do not claim WCAG compliance without sufficient evidence. Report actual tested criteria.

---

# 9. FRONTEND VERIFICATION — MUST BE FRESH

Use the repository's current scripts/dependencies; inspect `web/package.json` rather than inventing commands.

At minimum run fresh equivalents of:

```bash
cd web
./node_modules/.bin/eslint src
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/vitest run
node scripts/build.mjs
```

If package scripts provide canonical wrappers, run them too.

Acceptance:

```text
ESLint: 0 errors
TypeScript: 0 errors
Vitest: 0 failures
Web build: exit 0
```

Run browser E2E for Workspace flows after unit/component tests.

Verify the Web build copies the expected embeddable artifacts to the Go Web dist directory and that no stale pre-redesign bundle remains embedded.

---

# 10. BACKEND / GO 1.25 VERIFICATION — MANDATORY

The handoff environment had Go 1.23.2 and could not download Go 1.25 because outbound network was blocked.

Use **Go 1.25+** and run fresh:

```bash
go version
go vet ./...
go test ./...
go test -race ./...
```

Where race is unsupported on a platform, record that explicitly and run it on Linux at minimum.

Investigate/fix any failures. Do not merely report that they existed before the Workspace OS work if they affect current correctness.

Also revisit the high-risk backend items from the prior independent audit and prove their current state from code + tests:

1. Agent starts in the Project's canonical workspace, not server cwd.
2. Nexus Project/Agent write APIs enforce CSRF appropriately.
3. Stop only reports terminal state after the provider/process has actually stopped or has a truthful failure state.
4. Deleting Project/Agent cannot leave an untracked live provider/runtime behind.
5. Recovery/listing exposes effective/recoverable state truthfully.
6. Real provider/profile allocation replaces fake/default in real user flows.
7. Cross-platform Nexus data directory uses the canonical config/data-dir policy.
8. Reconfiguration has compensation/rollback on launch/persist failure.
9. Origin/private/VPN rules are internally consistent and public exposure remains refused by default.
10. Agent terminal broker follows current RuntimeGeneration correctly instead of binding permanently to the generation that was current at initial attach.

For every item state:

```text
PASS — evidence
FAIL — evidence + fix
NOT SUPPORTED — honest reason
```

Do not fabricate provider-level verified resume.

---

# 11. REAL INTEGRATION / E2E SCENARIOS

Start the real Nexus server and test through the browser/API, not only isolated components.

At minimum cover:

### Projects

```text
create Project
reload Nexus
same ProjectID
same project selected where appropriate
layout persists
project switching does not cross-contaminate layout/Agents
```

### Persistent Agent

```text
create/start Agent
open Agent terminal
close/reopen visual terminal surface
Agent remains alive
close browser/reopen browser
reattach to same live Agent/runtime when applicable
```

### Workspace terminal

```text
open terminal
drag to another stack
split
resize
maximize/restore
switch tabs
return to terminal
no extra provider process created
terminal remains usable
```

### Runtime generation / reconfigure

For an option requiring restart:

```text
same AgentID
new RuntimeGeneration
truthful continuity state
terminal surface survives/reconnects according to broker contract
rollback or clear failure on reconfigure failure
```

### Resources

Verify Resource page uses real provider/profile/quota/health state and explanation, not decorative numbers.

### Maestro

Verify current Maestro integration/fallback/degraded behavior truthfully. If Maestro functionality is still partial, label it partial rather than faking advice.

### Missions

Verify current Mission/Plan surfaces against existing backend capability. Do not visually imply durable autonomous Mission Runner behavior if that engine is not yet implemented.

This last rule is critical: future-planned Intelligence/WorkPlan/Mission Runner features may be shown as preview/planning surfaces, but UI state must never claim autonomous execution that the backend cannot yet perform.

---

# 12. THEMES + RESPONSIVE MATRIX

Run representative flows under:

```text
System
Dark
Light
High Contrast
```

At minimum validate desktop and mobile for Dark and Light, and desktop keyboard/focus for High Contrast.

Check accents do not break contrast.

Test reduced motion.

Test Compact and Comfortable densities.

---

# 13. PERFORMANCE / STABILITY

Perform practical checks for:

- repeated open/close tabs does not leak uncontrolled listeners;
- terminal `ResizeObserver` disconnects correctly;
- rapid split resize does not crash;
- rapid project switching does not save layout into wrong Project;
- slow terminal observer does not block runtime;
- large output remains bounded according to existing terminal/session policy;
- command palette scales with many Agents;
- no repeated API polling caused by unstable React dependency arrays;
- no serious console errors/warnings in normal flows.

Profile only enough to find real issues; do not turn finalization into speculative optimization.

---

# 14. SECURITY CHECK

Freshly inspect:

- CSP/security headers;
- session expiration/rotation/logout where currently required;
- CSRF;
- Origin checks;
- WebSocket auth;
- no wildcard CORS;
- no secrets in Project/Agent config/events/layout/prompt/demo data;
- path canonicalization and symlink traversal;
- Agent/Project IDOR isolation;
- demo mode cannot mutate real state;
- popout surface URL must not serialize secrets.

Create/update:

```text
DEV/NEXUS_WORKSPACE_OS_SECURITY_QA.md
```

Critical security gaps are NO-GO.

---

# 15. CROSS-PLATFORM RUNTIME EVIDENCE

Nexus support claims must distinguish compile evidence from runtime evidence.

Required runtime matrix before claiming full 1.0 platform support:

### Linux

```text
UDS
PTY
Agent start/stop
terminal attach/reattach
Workspace terminal
reconfigure/recovery
```

### Windows

```text
Named Pipe
ConPTY
resize
Agent start/stop
terminal attach/reattach
Workspace terminal
reconfigure/recovery
```

### macOS

```text
UDS
PTY
Agent start/stop
terminal attach/reattach
Workspace terminal
reconfigure/recovery
```

If the current machine cannot execute all three OSes, use CI/real machines where available and report missing runtime proof as CONDITIONAL_GO rather than inventing support evidence.

---

# 16. REQUIRED FINAL REPORTS

Produce/update at minimum:

```text
DEV/NEXUS_WORKSPACE_OS_FINAL_QA.md
DEV/NEXUS_WORKSPACE_OS_VISUAL_QA.md
DEV/NEXUS_WORKSPACE_OS_ACCESSIBILITY.md
DEV/NEXUS_WORKSPACE_OS_SECURITY_QA.md
DEV/NEXUS_WORKSPACE_OS_PLATFORM_MATRIX.md
DEV/NEXUS_WORKSPACE_OS_FINAL_HANDOFF.md
```

`FINAL_QA` must contain a requirement-by-requirement matrix:

```text
Requirement
Status
Evidence
Test/command
Known limitation
```

Explicitly compare:

```text
approved Workspace OS spec
vs
actual rendered UI
vs
actual backend behavior
```

Do not use repository documentation as evidence for behavior unless backed by code/test/runtime proof.

---

# 17. RELEASE VERDICT

Return exactly one:

```text
GO
CONDITIONAL_GO
NO_GO
```

### GO

Only when Workspace OS core UX, integration, security, frontend checks, Go checks and required runtime evidence are actually green.

### CONDITIONAL_GO

Allowed when core behavior is correct but one non-critical external platform/runtime evidence item remains unavailable, and the missing evidence is explicitly identified.

### NO_GO

Required for any of:

- Agent/process killed merely by closing a Workspace tab;
- terminal cross-talk between Agents/Projects;
- layout from one Project leaking into another;
- terminal broker silently attaching to wrong runtime generation;
- false STOPPED or false VERIFIED continuity;
- Workspace visual shell fundamentally unusable at desktop/mobile;
- critical accessibility keyboard trap;
- critical security bypass;
- broken CSRF/Origin/WS auth;
- Go/backend regression affecting Project/Agent lifecycle;
- build/test failure;
- stale embedded Web bundle;
- claims of functionality not actually implemented.

---

# 18. GIT / DELIVERY RULES

You are authorized to:

- create local worktree/branch;
- create local commits;
- fix code/tests/docs;
- rerun verification;
- generate local screenshots/reports/build artifacts.

Do NOT without explicit human approval:

- push to remote;
- merge to `main`;
- create public release/tag;
- deploy production;
- publish package;
- introduce paid services;
- delete user data.

At completion provide:

```text
Branch
Final commit
Dirty/clean working tree
Commands executed
Frontend result
Go result
Browser/E2E result
Accessibility result
Security result
Platform result
Verdict
Remaining limitations
```

If the harness can create a ZIP, produce a final ZIP preserving `.git` and source, excluding only disposable caches when appropriate.

---

# 19. AUTOPILOT STOP CONDITIONS

Do not stop because one test fails. Diagnose and fix it.

Do not stop because browser QA finds visual defects. Fix them and rerun screenshots.

Do not stop because a provider adapter differs. Inspect its declared capabilities and implement truthful behavior.

Stop and request the user only when:

- a new secret/account/login is genuinely required;
- a paid external service must be introduced;
- a destructive external action is needed;
- a product decision contradicts the approved spec and cannot be inferred safely;
- required OS hardware/runner does not exist and cannot be reached, after completing everything else possible.

Otherwise continue until the verdict is justified by fresh evidence.

---

# FINAL OBJECTIVE

When this prompt finishes, IAPro Nexus must no longer merely *compile as a Workspace OS*. It must have evidence that it **renders, behaves and integrates as the approved Workspace OS** while keeping Persistent Agent/runtime/resource/Maestro functionality truthful.

The target experience is:

```text
Projects feel like workspaces.
Agents feel persistent.
Terminals feel native to the workspace.
Tabs/splits/popouts feel productive rather than decorative.
Maestro/Resources/Missions feel integrated rather than separate admin pages.
The UI is themeable, accessible and understandable.
Closing UI never silently means killing work.
Every backend state shown in the UI is truthful.
```

Finish that product. Do not merely write another plan for it.
