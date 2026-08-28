# Nexus V1 — Component Inventory

See `NEXUS_V1_UI_ARCHITECTURE.md` for structure. Current reusable components:

| Component | Location | Reused by |
|---|---|---|
| `Button` (tones/sizes) | `web/src/ui/primitives.tsx` | ProjectsPage (Add/Create/Start/Stop/Terminal) |
| `Badge` (tones) | same | project mode, agent status |
| `Card` | same | project rail, agent rows |
| `Input` | same | path, agent name, command palette |
| `Spinner` / `EmptyState` | same | loading / no-projects / no-agents |
| `AppShell` + nav model | `web/src/nexus/AppShell.tsx` | App |
| `CommandPalette` | same | App (Ctrl/Cmd+K) |
| `ProjectsPage` | `web/src/nexus/ProjectsPage.tsx` | Overview/Projects/Agents views |
| `AgentTerminal` | `web/src/nexus/AgentTerminal.tsx` | ProjectsPage (agent-scoped xterm) |
| `nexus` API client | `web/src/nexus/api.ts` | ProjectsPage |
| `AgentStatusBadge` mapping | ProjectsPage helper | agent rows |

Pending gates: `StatusDescriptor`/`ActionDescriptor` (§174-175), `ResourcePicker`
(§176), shared agent-config engine (§177), Maestro recommendation card (§178),
`SplitPane`/`ResizablePanel`, `Drawer`/`ConfirmDialog` (§74), Storybook previews.

Reusability rule (§173): no provider-specific components (no `CodexButton`); no
duplicated buttons/badges/forms introduced in this gate.
