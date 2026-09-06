# IAPro Nexus Workspace OS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the approved project-first Workspace OS frontend without regressing existing Nexus backend capabilities.

**Architecture:** Introduce a serializable workspace model, theme/design-system provider and typed workspace surfaces. Replace the page-centric App shell with a NexusShell + WorkspaceSurfaceHost, while existing product APIs remain the source of data and AgentID remains the terminal identity.

**Tech Stack:** React 19, TypeScript 5.9, Tailwind 4, xterm.js, Lucide, Vitest, esbuild.

**Spec:** `docs/superpowers/specs/2026-08-29-nexus-workspace-os-design.md`

## Global Constraints
- Preserve existing Project/Agent/Resources/Maestro/Missions backend work from the uploaded ZIP.
- Closing a terminal tab must not stop the Agent.
- No false claims of provider continuity or runtime state.
- No external dependency may be required to build this offline handoff.
- Theme, accessibility and responsive behavior are first-class.

---

### Task 1: Workspace model and persistence
**Files:** `web/src/workspace/model.ts`, `web/src/workspace/model.test.ts`, `web/src/workspace/state.ts`, `web/src/workspace/state.test.ts`
- [ ] RED: tests for open/focus/split/move/close/collapse/maximize/ratio/restore.
- [ ] GREEN: implement typed serializable model and reducer.
- [ ] Verify focused tests.

### Task 2: Design system and themes
**Files:** `web/src/design-system/*`, `web/src/index.css`
- [ ] RED: tests for scheme/accent resolution and persisted preferences.
- [ ] GREEN: ThemeProvider, tokens and reusable primitives.
- [ ] Verify focused tests and typecheck.

### Task 3: Command palette and tour
**Files:** `web/src/app/commands/*`, `web/src/app/tour/*`
- [ ] RED: test search/ranking and tour filtering/progression.
- [ ] GREEN: accessible command palette and guided tour model/view.

### Task 4: Workspace renderer
**Files:** `web/src/workspace/*`
- [ ] Implement recursive stacks/splits, draggable tabs, keyboard resize, maximize, popout and taskbar.
- [ ] Preserve layout in Project storage.

### Task 5: Project-first shell and product surfaces
**Files:** `web/src/app/*`, `web/src/features/*`, `web/src/nexus/*`
- [ ] Build Project Rail/topbar/project nav/taskbar.
- [ ] Convert Overview/Work/Agents/Resources/Maestro/Missions/Sessions/Settings to surfaces.
- [ ] Keep legacy runtime/provider/events reachable.
- [ ] Make AgentTerminal resize with its pane and use AgentID WebSocket URL.

### Task 6: Offline production build and cleanup
**Files:** `web/scripts/build.mjs`, `web/package.json`, `web/index.html`, embedded dist.
- [ ] Replace Bun-only build with Node+esbuild+Tailwind using available dependencies.
- [ ] Run lint/typecheck/tests/build.
- [ ] Package source + `.git` and final local-validation prompt.
