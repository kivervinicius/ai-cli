# Nexus V1 — UI Architecture & Component Inventory

Charter §69-88. Web-first, component-first, design-system-driven.

## Structure (converging; App.tsx no longer a god component)

```text
web/src/
  ui/            primitives.tsx · tokens (index.css :root vars)
  nexus/         AppShell.tsx · ProjectsPage.tsx · AgentTerminal.tsx · api.ts
  components/    legacy runtime views (Dashboard, TerminalView, ProvidersView, …)
  api.ts types.ts index.tsx
```

## Design tokens (dark primary, light/system prepared)

CSS custom properties in `:root`: `--nx-bg/surface/surface-2/border/text/muted`,
`--nx-brand/brand-2/success/warning/danger/info`, `--nx-radius-*`, `--nx-space-*`,
`--nx-font-mono` (§78). Dark is the primary theme but tokens keep light/system reachable.

## Primitives (§71) — implemented

`Button` (tones+sizes) · `Badge` (tones) · `Card` · `Input` · `Spinner` · `EmptyState`

## Layout (§72)

`AppShell` (sidebar nav Nexus/Legacy + topbar + command-palette trigger) · `CommandPalette`
(`Ctrl/Cmd+K` jump to nav, §83). `SplitPane/ResizablePanel` pending Gate 4.

## Feedback & overlays (§73-74)

`EmptyState`/`Spinner` implemented; `Alert/Toast/Skeleton/StatusIndicator`,
`Modal/Drawer/ConfirmDialog` per design rules (Modal=confirmation, Drawer=config,
Full page=workflows, Popover=quick actions) as gates land.

## Domain components (§75-76)

`AgentStatusBadge` (status→tone), Project card, Agent card, AgentTerminal frame.
Provider-specific components are avoided — generic `ProviderBadge`/`ResourcePicker`
to come with Gate 5.

## Rules applied

- `key={agentID}` for terminals, never `key={runtimeID}` (§31).
- Actions/status via descriptors, no inconsistent colors (§174-175).
- Loading/empty/error/stale states everywhere (§85); no fake data in production (§106).
- Accessibility from Gate 1: semantic buttons, ARIA labels on icon buttons, kbd hints.

## Component inventory (this gate)

| Component | File | Notes |
|---|---|---|
| Button/Badge/Card/Input/Spinner/EmptyState | ui/primitives.tsx | tokens-based |
| AppShell + NEXUS_NAV + CommandPalette | nexus/AppShell.tsx | Ctrl/Cmd+K |
| ProjectsPage | nexus/ProjectsPage.tsx | project rail + agents + terminal |
| AgentTerminal | nexus/AgentTerminal.tsx | xterm, agent-scoped WS |
| nexus API client | nexus/api.ts | CSRF-aware |

## Reusability gate (§173)

ProjectsPage renders agents via a single AgentCard pattern; actions share
`Button` tones; no duplicated badges/forms introduced. Storybook-equivalent
(component state previews) is a Gate 8 polish item.
