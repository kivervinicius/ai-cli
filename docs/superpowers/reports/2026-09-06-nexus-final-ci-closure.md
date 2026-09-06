# IAPro Nexus Final CI Closure — 2026-09-06

## Identity

- Initial branch: `feat/nexus-maximum-delivery`
- Initial SHA: `4570afca8434bc181e4a64831dc9365bd020867b`
- Fix branch: `fix/final-ci-closure`
- Final SHA: `08432558f911b859f01a7f1beae10b5c1bc1290d`
- Target branch: `feat/nexus-maximum-delivery`
- Target SHA: `4570afca8434bc181e4a64831dc9365bd020867b`
- CI run ID: `34008087907` (baseline only; failed)
- Latest fix-branch CI run: none; branch was not pushed.

## Findings matrix

| ID | Finding | Initial | Root cause / change | Final |
| --- | --- | --- | --- | --- |
| FC-01 | `deeplink.go` gofmt | STILL_PRESENT | Formatted file | RESOLVED |
| FC-02 | golangci action v2 compatibility | STILL_PRESENT | Updated action to v7 and pinned Go | RESOLVED locally; CI pending |
| FC-03 | build/install conflation | STILL_PRESENT | `build` now emits artifact; `install-local` installs | RESOLVED |
| FC-04 | developer-home hardcode | STILL_PRESENT | Portable `HOME`/workspace-relative paths; legacy imports fixed | RESOLVED operationally; historical docs remain |
| FC-05 | browser blocked before assertions | STILL_PRESENT | Portable build and authenticated route handling | RESOLVED locally |
| FC-06 | macOS WorkspaceStore identity | STILL_PRESENT | Consumer uses existing `PathRef` filesystem identity | RESOLVED in code; native pending |
| FC-07 | macOS worktree identity | STILL_PRESENT | Same identity-aware consumer path | RESOLVED in code; native pending |
| FC-08 | Windows identity unavailable | STILL_PRESENT | Native final-path handle resolution | RESOLVED in code; native pending |
| FC-09 | Windows short/long workspace path | STILL_PRESENT | Native identity avoids fixed 8.3 strings | RESOLVED in code; native pending |
| FC-10 | Windows worktree identity | STILL_PRESENT | Reuses identity-aware equivalence | RESOLVED in code; native pending |
| FC-11 | Windows SessionHost named pipe | STILL_PRESENT | ConPTY success-path fix; transport native execution not available | STILL_PRESENT / unverified |
| FC-12 | listener failure fixture | STILL_PRESENT | No native runner to prove lifecycle semantics | STILL_PRESENT / unverified |
| FC-13 | RuntimeLauncher handshake | STILL_PRESENT | No native Windows execution available | STILL_PRESENT / unverified |
| FC-14 | ConPTY echo | STILL_PRESENT | Corrected inverted `CreatePseudoConsole` HRESULT branch | RESOLVED in code; native pending |
| FC-15 | ConPTY interactive | STILL_PRESENT | Same correction | RESOLVED in code; native pending |
| FC-16 | Windows FSMkdir | STILL_PRESENT | No native Windows execution available | STILL_PRESENT / unverified |
| FC-17 | Windows listen state | STILL_PRESENT | No native Windows execution available | STILL_PRESENT / unverified |
| FC-18 | Windows providers JSON | STILL_PRESENT | No native Windows execution available | STILL_PRESENT / unverified |
| FC-19 | Secret Service boundary | STILL_PRESENT | Linux-only implementation/tests; truthful Darwin/Windows unsupported paths | RESOLVED in code; native pending |
| FC-20 | Codex attribution Windows | STILL_PRESENT | No native Windows execution available | STILL_PRESENT / unverified |
| FC-21 | Codex attribution macOS | STILL_PRESENT | No native macOS execution available | STILL_PRESENT / unverified |
| FC-22 | desktop build tags | STILL_PRESENT | Linux-only WebKit tag selection | RESOLVED in code; native pending |
| FC-23 | desktop native CI proof | STILL_PRESENT | Added Linux/Windows/macOS Wails jobs | RESOLVED in workflow; CI pending |
| FC-24 | security Go patch parity | STILL_PRESENT | Standardized 1.25.14 | RESOLVED locally; CI pending |
| FC-25 | GoReleaser snapshot blocked | STILL_PRESENT | Snapshot now passes locally on final SHA | RESOLVED locally; CI pending |

## Commits

- `11ec21d` — format desktop deep-link implementation.
- `7ea730d` — separate build from local installation and align toolchain/action.
- `bb77d23` — use filesystem identity for workspace equivalence.
- `aae3d09` — correct Windows ConPTY success-path handling.
- `b8217cc` — scope Secret Service isolation to Linux.
- `b8f6b68` — add native Wails desktop build lanes and durable ledger.
- `c7f456e` — preserve authenticated deep-link navigation and route hydration.
- `236f950` — synchronize embedded frontend bundle.
- `9c6c241` — remove environment-specific Playwright imports and E2E timing waits.
- `0843255` — format workspace route synchronization.

## RED/GREEN evidence

- RED: `gofmt -l .` listed `internal/desktop/deeplink.go`; GREEN: same check returned no files.
- RED: baseline `make build` attempted developer-home installation; GREEN: `make build` exited 0 and only produced checkout-local `./nexus`.
- RED: baseline browser gate failed before assertions; GREEN: `npm run test:e2e` exited 0 after Axe, deep-links, breakpoints and settings assertions.
- RED: browser Axe found serious contrast violations; GREEN: same gate reported zero serious/critical violations.
- RED: workspace alias test failed before identity-aware consumer fix; GREEN: targeted workspace tests and `go test -count=10` passed.
- RED: ConPTY path had inverted HRESULT branch; GREEN: Windows terminal/host packages cross-compiled and Linux suite passed. Native Windows execution remains unproven.
- GREEN: `go vet ./...`, `go test -race ./...`, `make lint-go`, `make security`, `make quality`, `npm --prefix web run quality` all exited 0 on final SHA.
- GREEN: `goreleaser release --snapshot --clean` exited 0; `sha256sum -c dist/checksums.txt` returned OK for all 10 entries.

## Platform matrix

| Capability | Linux | Windows | macOS |
| --- | ---: | ---: | ---: |
| Build | PASS | cross-compile only | cross-compile only |
| Unit | PASS | cross-compile/`-exec=true` only | cross-compile/`-exec=true` only |
| Race | PASS | NOT RUN | NOT RUN |
| PTY/ConPTY | PASS | NOT RUN natively | cross-compiled only |
| SessionHost | PASS | NOT RUN natively | cross-compiled only |
| RuntimeLauncher | PASS | NOT RUN natively | cross-compiled only |
| PathIdentity | PASS | cross-compiled only | cross-compiled only |
| Workspace | PASS | cross-compiled only | cross-compiled only |
| Worktree | PASS | cross-compiled only | cross-compiled only |
| Credential isolation | PASS (Secret Service) | unsupported path compiled | unsupported path compiled |
| Codex Usage | PASS locally | NOT RUN natively | NOT RUN natively |
| Web | PASS | NOT RUN natively | NOT RUN natively |
| Desktop | PASS (`make build-desktop`) | NOT RUN natively | NOT RUN natively |
| Security | PASS | CI pending | CI pending |

## Browser

- E2E: PASS.
- Axe: PASS; zero serious/critical violations, one minor violation.
- Visual: PASS as part of the six viewport screenshot smoke assertions.

## Desktop

- Linux: PASS locally with `make build-desktop`.
- Windows: workflow lane added, not executed.
- macOS: workflow lane added, not executed.

## Release

- Snapshot: PASS locally at final SHA.
- Targets: Linux amd64/arm64, Darwin amd64/arm64, Windows amd64/arm64.
- Checksums: 10/10 validated `OK`.
- Artifacts: 6 archives and 4 Linux packages produced.

## Skips

- `internal/release/installer_test.go`: `LEGITIMATE_PLATFORM_SPECIFIC` (Unix shell syntax).
- Unix wrapper tests in `internal/nexus/autonomyguard`: `LEGITIMATE_PLATFORM_SPECIFIC`.
- Git/curl unavailable fixtures: `OPTIONAL_EXTERNAL_INTEGRATION`.
- `internal/runtime/secret_service_linux_test.go`: `OPTIONAL_EXTERNAL_INTEGRATION` when `dbus-run-session` is absent.
- No suspicious skip was added by this campaign.
- Native Windows/macOS jobs were not skipped; they were not executed because the fix branch was not pushed and this environment has no native runners. This is missing evidence, not a legitimate skip.

## Final verdict

**NO-GO**

The repository is locally green on the fix SHA and the release snapshot succeeds, but the required same-SHA GitHub Actions run, native Windows runtime/ConPTY/Named Pipe tests, native macOS tests, and native Wails desktop builds were not executed. The target branch remains at the baseline failing SHA.
