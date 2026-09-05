# Frontend verification report

- Generated: `2026-09-05T05:51:08Z`
- Branch: `feat/nexus-maximum-delivery` @ `4b2b267`
- Verdict: **FAIL** (6 pass / 2 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| TypeScript (`tsc --noEmit`) | yes | FAIL | 6747ms | exit 2<br>src/workspace/WorkspaceRenderer.tsx(745,22): error TS2304: Cannot find name 'Plus'. |
| ESLint (`eslint src`) | yes | FAIL | 4255ms | exit 1<br>/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/src/features/settings/SettingsSurface.tsx<br>   24:3   warning  'Select' is defined but never used. Allowed unused vars must match /^_/u                                            @typescript-eslint/no-unused-vars<br>   81:27  warning  'setCheckingUpdates' is assigned a value but never used. Allowed unused vars must match /^_/u                       @typescript-eslint/no-unused-vars<br>  173:5   warning  React H |
| Null-safe API array access | yes | PASS | 107ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 6193ms | ✓ src/features/work/directSessionModel.test.ts (7 tests) 17ms<br> ✓ src/design-system/primitives/ContextMenu.test.ts (1 test) 8ms<br> ✓ src/features/work/flowRunModel.test.ts (3 tests) 7ms<br> ✓ src/features/agents/terminalSkillsAndAlias.test.ts (3 tests) 4ms<br> ✓ src/services/WorkspaceLayoutService.test.ts (5 tests) 7ms<br> ✓ src/app/sessionModel.test.ts (2 tests) 4ms<br> ✓ src/workspace/state.test.ts (8 tests) 12ms<br> ✓ src/lib/safeArray.test.ts (3 tests) 16ms<br> ✓ src/keyboard/KeyboardShor |
| i18n catalog parity | yes | PASS | 1188ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 6ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  01:51:26<br>   Duration  663ms (transform 176ms, setup 0ms, collect 209ms, tests 6ms, environment 0ms, prepare 147ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 920ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 228ms<br><br>  dist/bundle.js  1.1mb ⚠️<br><br>⚡ Done in 362ms |
| Embed sync (web/dist ≡ internal/.../dist) | yes | PASS | 2ms | bundles idênticos (1180761 bytes) |
| Critical UI markers in bundle | yes | PASS | 6ms | marcadores críticos presentes (5) |

## Residual risks / next operator steps

- Hard gates failed — do not claim frontend delivery until green.
  - Fix `typecheck` then re-run `make web-verify`.
  - Fix `lint` then re-run `make web-verify`.

### Dirty paths

```
M internal/control/web/dist/bundle.css
 M internal/control/web/dist/bundle.js
 M web/eslint.config.js
 M web/package.json
 M web/src/App.tsx
 M web/src/api.test.ts
 M web/src/api.ts
 M web/src/app/NexusDemoApp.tsx
 M web/src/app/NexusShell.tsx
 M web/src/app/NexusWorkspaceApp.tsx
 M web/src/app/WorkspaceSurfaceHost.tsx
 M web/src/app/attentionRadarModel.test.ts
 M web/src/app/attentionRadarModel.ts
 M web/src/app/commands/CommandPalette.tsx
 M web/src/app/commands/registry.test.ts
 M web/src/app/commands/registry.ts
 M web/src/app/components/LanguagePicker.tsx
 M web/src/app/documentTitle.test.ts
 M web/src/app/documentTitle.ts
 M web/src/app/maestroHonesty.test.ts
 M web/src/app/modals/MaestroControlModal.tsx
 M web/src/app/modals/WelcomeModal.tsx
 M web/src/app/projectSelection.test.ts
 M web/src/app/projectSelection.ts
 M web/src/app/runtimeAgentMapping.test.ts
 M web/src/app/sessionModel.ts
 M web/src/app/surfaces.test.ts
 M web/src/app/surfaces.ts
 M web/src/app/tour/ProductTour.tsx
 M web/src/app/tour/tour.test.ts
 M web/src/app/tour/tour.ts
 M web/src/app/useNexusData.ts
 M web/src/app/versionHonesty.test.ts
 M web/src/app/workspace-os.css
 M web/src/app/workspaceMissionRoute.test.ts
 M web/src/components/AttentionNotification.test.ts
 M web/src/components/AttentionNotificationCard.tsx
 M web/src/components/AttentionNotificationManager.tsx
 M web/src/components/ContinueModal.tsx
 M web/src/components/Dashboard.tsx
 M web/src/components/EventsView.tsx
 M web/src/components/GlobalAttentionRadar.tsx
 M web/src/components/HandoffModal.tsx
 M web/src/components/ProvidersView.tsx
 M web/src/components/Sidebar.tsx
 M web/src/components/StartModal.tsx
 M web/src/components/TerminalPane.tsx
 M web/src/components/TerminalView.tsx
 M web/src/components/attentionText.test.ts
 M web/src/components/terminalViewModel.test.ts
 M web/src/components/terminalViewModel.ts
 M web/src/design-system/primitives/ContextMenu.tsx
 M web/src/design-system/primitives/index.tsx
 M web/src/design-system/theme/ThemeProvider.tsx
 M web/src/design-system/theme/theme.test.ts
 M web/src/design-system/theme/theme.ts
 M web/src/design-system/theme/themePresets.ts
 M web/src/features/agents/AgentConfigurationSurface.tsx
 M web/src/features/agents/AgentsSurface.tsx
 M web/src/features/agents/AskAgentDialog.tsx
 M web/src/features/agents/NewAgentModal.tsx
 M web/src/features/agents/askAgentModel.test.ts
 M web/src/features/agents/terminalSkillsAndAlias.test.ts
 M web/src/features/overview/ProjectOverviewSurface.tsx
 M web/src/features/overview/overviewRecover.test.ts
 M web/src/features/projects/AddProjectModal.tsx
 M web/src/features/projects/BranchSwitcherModal.tsx
 M web/src/features/projects/DirectoryBrowserModal.tsx
 M web/src/features/projects/ProjectCreateActions.tsx
 M web/src/features/projects/ProjectHub.tsx
 M web/src/features/projects/ProjectManagerSurface.tsx
 M web/src/features/projects/ProjectRail.tsx
 M web/src/features/projects/ProjectScanModal.tsx
 M web/src/features/projects/projectRail.test.ts
 M web/src/features/sessions/SessionsSurface.tsx
 M web/src/features/settings/IntelligenceProviderCombo.tsx
 M web/src/features/settings/SettingsSurface.tsx
 M web/src/features/settings/intelligenceProfiles.test.ts
 M web/src/features/settings/intelligenceProfiles.ts
 M web/src/features/work/ComposerSurface.tsx
 M web/src/features/work/DirectSessionLauncher.tsx
 M web/src/features/work/FlowCanvas.tsx
 M web/src/features/work/FlowRunSurface.tsx
 M web/src/features/work/FlowRunsHistorySurface.tsx
 M web/src/features/work/FlowStepInspector.tsx
 M web/src/features/work/MissionAutonomyCard.tsx
 M web/src/features/work/PlanBuilderSurface.tsx
 M web/src/features/work/WorkSurface.tsx
 M web/src/features/work/clarificationModel.test.ts
 M web/src/features/work/composerModel.test.ts
 M web/src/features/work/composerModel.ts
 M web/src/features/work/composerSessionModel.test.ts
 M web/src/features/work/composerSessionModel.ts
 M web/src/features/work/directSessionModel.test.ts
 M web/src/features/work/directSessionModel.ts
 M web/src/features/work/flowModel.test.ts
 M web/src/features/work/flowModel.ts
 M web/src/features/work/flowRunModel.ts
 M web/src/features/work/planBuilderModel.test.ts
 M web/src/features/work/planBuilderModel.ts
 M web/src/features/work/planBuilderScheduling.test.ts
 M web/src/i18n/i18n.test.ts
 M web/src/i18n/index.ts
 M web/src/i18n/resources.ts
 M web/src/keyboard/KeyboardShortcutRegistry.test.ts
 M web/src/keyboard/KeyboardShortcutRegistry.ts
 M web/src/nexus/AgentTerminal.tsx
 M web/src/nexus/MaestroPage.tsx
 M web/src/nexus/MissionsPage.tsx
 M web/src/nexus/ResourcePicker.tsx
 M web/src/nexus/TerminalActionDialog.tsx
 M web/src/nexus/agentRecover.test.ts
 M web/src/nexus/agentRecover.ts
 M web/src/nexus/agentTerminalModel.test.ts
 M web/src/nexus/agentTerminalModel.ts
 M web/src/nexus/api.test.ts
 M web/src/nexus/api.ts
 M web/src/nexus/terminalProtocol.test.ts
 M web/src/nexus/terminalProtocol.ts
 M web/src/nexus/workPlan.ts
 M web/src/notifications/InAppNotificationCenter.tsx
 M web/src/notifications/attentionDelivery.test.ts
 M web/src/notifications/attentionPushCopy.test.ts
 M web/src/notifications/inAppNotificationModel.test.ts
 M web/src/notifications/notificationModel.ts
 M web/src/notifications/notificationPrefs.ts
 M web/src/services/WorkspaceLayoutService.test.ts
 M web/src/services/WorkspaceLayoutService.ts
 M web/src/types.ts
 M web/src/workspace/PtyLiveChromeContext.tsx
 M web/src/workspace/WorkspacePresentationProvider.tsx
 M web/src/workspace/WorkspaceProvider.tsx
 M web/src/workspace/WorkspaceRenderer.tsx
 M web/src/workspace/WorkspaceTaskbar.tsx
 M web/src/workspace/arrange.test.ts
 M web/src/workspace/arrange.ts
 M web/src/workspace/arrangePresets.test.ts
 M web/src/workspace/arrangePresets.ts
 M web/src/workspace/model.test.ts
 M web/src/workspace/model.ts
 M web/src/workspace/presentation.test.ts
 M web/src/workspace/presentation.ts
 M web/src/workspace/ptyLiveChrome.test.ts
 M web/src/workspace/ptyLiveChrome.ts
 M web/src/workspace/state.test.ts
 M web/src/workspace/state.ts
 M web/src/workspace/surfaceAttention.test.ts
 M web/src/workspace/surfaceAttention.ts
 M web/src/workspace/taskbarHonesty.test.ts
?? web/.husky/
?? web/.prettierignore
?? web/.prettierrc
?? web/.stylelintrc.json
?? web/src/features/projects/ProjectCreateMenu.tsx
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
