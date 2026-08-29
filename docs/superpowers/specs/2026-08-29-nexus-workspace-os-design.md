# IAPro Nexus Workspace OS — Design Specification

## Goal
Replace the page-centric Nexus web dashboard with a project-first, OS-like workspace that keeps terminals and product surfaces open as movable tabs/splits while preserving the existing Nexus backend and future Intelligence/WorkPlan/Mission Runner architecture.

## Product principles
- Project is the primary context.
- Agent is persistent identity; Runtime is ephemeral implementation detail.
- Closing a workspace surface never stops an Agent.
- Terminal surfaces are keyed by AgentID.
- The center of the app is a workspace, not a route/page.
- A user may work with tabs, horizontal/vertical splits, maximize and pop-out windows.
- Project layout is persisted and restored.
- Mobile keeps one active workspace stack while retaining the desktop layout model.
- The interface must support keyboard operation, visible focus, high contrast and reduced motion.
- Existing Resources, Maestro, Missions, Agent Configuration and legacy runtime capabilities remain reachable.

## Information architecture
### Global rail
Projects, Home, Resources, Maestro, Sessions, Settings.

### Project navigation
Overview, Work, Plan, Agents, Workspace, Maestro, Usage, Events.

### Workspace surfaces
Project overview, Work composer, Agents, Agent configuration, Agent terminal, Resources, Maestro, Missions/Plan, Sessions, Settings, legacy runtime/provider/event screens.

## Workspace model
A serializable tree of `stack` and `split` nodes. Stacks own tabs. Splits own two children and a ratio. Tabs reference a typed surface descriptor. The model supports open/focus, split, move, close/collapse, maximize and persistent restore.

## Theme and accessibility
Four schemes: dark, light, system, high-contrast. Four accents: purple, blue, cyan, neutral. User-selectable density and reduced-motion. Focus-visible, skip link, landmark regions, keyboard-resizable splitters, dialog focus handling and non-color status labels are required.

## Guided tour
The first-run tour explains Project Rail, Workspace, Agents, terminal persistence, Resources, Maestro, Work/Plan, command palette and personalization. It can be replayed from Help and only shows steps whose targets exist.

## Technology constraints
React 19, TypeScript, Tailwind CSS, xterm.js, Lucide. New online dependencies are not required for this handoff build. The workspace engine and tour are internal, typed components so they can later be swapped for FlexLayout/Radix/React Joyride without changing product contracts.

## Verification
TypeScript, ESLint, Vitest, production web build, generated embedded Go web assets. Visual/manual E2E and full Go 1.25 verification are explicitly delegated to the final local validation prompt if the container cannot provide browser/Go toolchain evidence.
