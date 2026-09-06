# IAPro Nexus Workspace OS — Handoff

## Status

This handoff contains the Project-first Workspace OS redesign implemented on top of the uploaded `ai-manageR(6).zip` continuation state.

Branch: `feat/nexus-workspace-os-handoff`

## Implemented

- Project-first Workspace OS shell
- Persistent Project rail and Project navigation
- Serializable Project workspace layout
- Tabs, nested horizontal/vertical splits, drag/move, resize, maximize/restore
- Keyboard-accessible split resizing
- Popout surfaces and taskbar
- Per-Project layout persistence
- Terminal surfaces addressed by stable AgentID
- Xterm refit via ResizeObserver when panes move/resize
- Design-system tokens and reusable primitives
- System, Dark, Light, High Contrast themes
- Purple, Blue, Cyan, Neutral accents
- Compact/Comfortable density and reduced-motion preference
- Command Palette (`Ctrl/Cmd+K`)
- Product tour and replay entry point
- Responsive Project rail / compact workspace behavior
- Work surface with Direct / Assisted / Planned entry modes
- Agents, Agent Configuration, Resources, Maestro, Plan/Missions, Sessions and Settings as Workspace surfaces
- Existing runtime/provider/events controls retained as advanced surfaces
- Synthetic `?demo=1` mode for UI/onboarding inspection without real project data
- Node + esbuild + Tailwind Web build, removing Bun as a build-time requirement
- Embedded Go Web distribution updated by the Web build

## Fresh verification performed in this handoff environment

From `web/`:

```bash
./node_modules/.bin/eslint src
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/vitest run
node scripts/build.mjs
```

Result at handoff time:

- ESLint: PASS
- TypeScript: PASS
- Vitest: PASS — 9 files / 36 tests
- Web build: PASS

## Verification not completed in this environment

### Go 1.25

The repository requires Go >= 1.25. The handoff container has Go 1.23.2. Automatic toolchain download is blocked by network policy, so backend tests could not be freshly rerun here.

Required local validation:

```bash
go version
go vet ./...
go test ./...
go test -race ./...
```

Use Go 1.25+.

### Rendered browser QA

The environment's Chromium is subject to an organization policy that blocks `127.0.0.1`, and the headless/Xvfb fallback did not provide a trustworthy rendered-page validation. Do not treat source/build checks as visual QA.

The finalization run must inspect the real application in a normal browser at desktop/tablet/mobile widths and exercise keyboard/accessibility behavior.

## Reference design

If present, `docs/design/reference-workspace-os-prototype.html` is the earlier Nexus layout prototype used as visual/product direction. The implementation is intentionally an evolution of that direction rather than a literal page-dashboard copy.

## Important finalization principle

Preserve the Workspace OS contracts and current backend continuation state. Finalization should diagnose and correct gaps, not reset the branch or rewrite the application from an older baseline.
