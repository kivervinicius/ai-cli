# Frontend verification report

- Generated: `2026-09-12T17:14:31Z`
- Branch: `feat/nexus-maximum-delivery` @ `b142171`
- Verdict: **FAIL** (9 pass / 1 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | FAIL | 6481ms | exit 1<br>Checking formatting...<br>[warn] src/wailsjs/wailsjs/go/desktop/App.d.ts<br>[warn] src/wailsjs/wailsjs/go/models.ts<br>[warn] src/wailsjs/wailsjs/runtime/package.json<br>[warn] src/wailsjs/wailsjs/runtime/runtime.d.ts<br>[warn] Code style issues found in 4 files. Run Prettier with --write to fix. |
| TypeScript (`tsc --noEmit`) | yes | PASS | 9779ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5338ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1197ms | ok |
| Null-safe API array access | yes | PASS | 95ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 9514ms | ✓ src/features/work/missionAutonomyModel.test.ts (1 test) 11ms<br> ✓ src/workspace/model.test.ts (14 tests) 19ms<br> ✓ src/workspace/arrange.test.ts (12 tests) 24ms<br> ✓ src/features/work/clarificationModel.test.ts (2 tests) 9ms<br> ✓ src/app/workspaceMissionRoute.test.ts (3 tests) 10ms<br> ✓ src/features/projects/projectRail.test.ts (4 tests) 25ms<br> ✓ src/lib/safeArray.test.ts (3 tests) 19ms<br> ✓ src/app/workspaceSurfaceStyles.test.ts (2 tests) 4ms<br> ✓ src/lib/networkHost.test.ts (3 tests |
| i18n catalog parity | yes | PASS | 1941ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 20ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  13:15:04<br>   Duration  1.12s (transform 443ms, setup 0ms, collect 550ms, tests 20ms, environment 0ms, prepare 166ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1579ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 236ms<br><br>  dist/bundle.js                                   343.4kb<br>  dist/chunks/chunk-OI6WT6YE.js                    277.9kb<br>  dist/chunks/FlowRunsHistorySurface-KU4XRBOS.js   228.3kb<br>  dist/chunks/chunk-KTQEXYIX.js                    159.0kb<br>  dist/bundle.css                                  157.2kb<br>  dist/chunks/chunk-5KKHRYCG.js  |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 0ms | bundles idênticos (351677 bytes) |
| Critical UI markers in bundle | yes | PASS | 2ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Hard gates failed — do not claim frontend delivery until green.
  - Fix `format` then re-run `make web-verify`.

### Dirty paths

```
M web/scripts/e2e-hardening-verify.mjs
 M web/src/api.ts
 M web/src/components/ErrorBoundary.tsx
 M web/src/features/work/AttentionCenter.module.scss
 M web/src/features/work/AttentionCenter.tsx
 M web/src/i18n/resources.ts
 M web/src/nexus/agentTerminalModel.test.ts
 M web/src/nexus/agentTerminalModel.ts
 M web/src/platform/desktopBridge.ts
 M web/src/wailsjs/wailsjs/go/desktop/App.d.ts
 M web/src/wailsjs/wailsjs/go/models.ts
 M web/src/wailsjs/wailsjs/runtime/package.json
 M web/src/wailsjs/wailsjs/runtime/runtime.d.ts
?? web/src/components/ErrorBoundary.module.scss
?? web/src/lib/networkHost.test.ts
?? web/src/lib/networkHost.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
