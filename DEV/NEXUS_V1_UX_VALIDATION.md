# Nexus V1 — UX Validation

Scope of this gate (Gate 1): web shell + project/agent vertical slice.

## Validated (real API, no fixtures)

- Navigation: Nexus (Overview/Projects/Agents/Resources/Maestro/Sessions/Settings) +
  Legacy (Runtimes/Providers/Events).
- Command palette: `Ctrl/Cmd+K` opens, filters nav, Enter navigates.
- Project lifecycle: add (path), select, MRU list, delete; empty state when none.
- Agent lifecycle: create, start (WORKING), stop, terminal; status badges; empty state.
- Agent terminal: xterm mounts, connects via agent-scoped WS, role indicator
  (CONTROL/VIEW_ONLY), disconnect message, detach button.
- Loading spinner; inline errors (red) for API failures.

## Honesty checks (§86)

- Start shows "Starting…" until the runtime returns RUNNING (no premature WORKING).
- Terminal shows disconnected state when WS closes; agent status only reflects
  confirmed store state.
- No fake buttons: "Terminal" is disabled until an agent has a RUNNING runtime.

## Pending gates

- Gate 3: configuration drawer + impact preview (apply-mode labels).
- Gate 4: split view, layout restore, browser close/reopen replay, multi-browser
  control transfer.
- Gate 8: accessibility audit (keyboard/ARIA/contrast), visual regression screenshots
  at 1440×900 / 1280×800 / 390×844, mobile minimum-operational, Storybook previews.
