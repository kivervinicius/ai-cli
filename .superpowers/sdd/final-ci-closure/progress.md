# Nexus Final CI Closure Progress

## Baseline

- Initial branch: `feat/nexus-maximum-delivery`
- Initial SHA: `4570afca8434bc181e4a64831dc9365bd020867b`
- Fix branch: `fix/final-ci-closure`
- Worktree: `.worktrees/fix-final-ci-closure`
- Remote branch: `origin/feat/nexus-maximum-delivery` at `4570afca8434bc181e4a64831dc9365bd020867b`
- Latest CI run: `34008087907`
- Latest CI SHA: `4570afca8434bc181e4a64831dc9365bd020867b`
- Latest CI status: failure
- CI log limitation: `gh run view --log-failed` returned HTTP 403 (repository admin rights required).

## Findings revalidation

| ID | Status at baseline | Evidence |
| --- | --- | --- |
| FC-01 | STILL_PRESENT | CI Linux Gofmt Check failed; source is `internal/desktop/deeplink.go`. |
| FC-02 | STILL_PRESENT | CI workflow uses `golangci/golangci-lint-action@v6` with golangci-lint v2.12.2. |
| FC-03 | STILL_PRESENT | `make build` writes to `LOCAL_BIN` and creates `ai` symlink. |
| FC-04 | STILL_PRESENT | Makefile operationally defaults to `/home/desenvolvedor/.local/bin`. |
| FC-05 | STILL_PRESENT | Browser job builds binary, then `Run browser verification` fails before browser assertions. |
| FC-06 | STILL_PRESENT | Requires local/native reproduction; prior finding remains unproven by current CI summary. |
| FC-07 | STILL_PRESENT | Requires local/native reproduction; prior finding remains unproven by current CI summary. |
| FC-08 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-09 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-10 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-11 | STILL_PRESENT | Current Windows `Test` job fails before dedicated transport lane. |
| FC-12 | STILL_PRESENT | Current Windows `Test` job fails before dedicated transport lane. |
| FC-13 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-14 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-15 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-16 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-17 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-18 | STILL_PRESENT | Requires Windows runner reproduction. |
| FC-19 | STILL_PRESENT | Prior macOS/Secret Service boundary finding; requires native reproduction. |
| FC-20 | STILL_PRESENT | Current Windows full test job fails before isolated test evidence. |
| FC-21 | STILL_PRESENT | Current macOS full race test job fails before isolated test evidence. |
| FC-22 | STILL_PRESENT | `build-desktop` unconditionally adds `webkit2_41`. |
| FC-23 | STILL_PRESENT | CI has no native Wails desktop build jobs. |
| FC-24 | STILL_PRESENT | Makefile security target pins Go 1.25.13 while CI uses 1.25. |
| FC-25 | STILL_PRESENT | Snapshot job is skipped because required platform jobs fail. |

## Work log

Updates are appended below after each reproduce/root-cause/fix/verification cycle.

## Cycles

### FC-06/07/08/09/10 — path identity consumers

- RED: `go test ./internal/control/workspace -run TestWorkspaceStoreRemoveResolvesFilesystemAliases -count=1` failed because an alias path was not found.
- Root cause: workspace consumer compared path text and did not use the existing `PathRef` identity.
- Change: consumer-level equivalence uses `FilesystemIdentity` when available and canonical text as fallback; Windows existing directories now use `GetFinalPathNameByHandle` through `x/sys/windows`.
- GREEN: targeted workspace tests passed, including `-count=10`; Windows config package cross-compiled successfully.

### FC-14/15 — ConPTY HRESULT handling

- RED: CI Windows native ConPTY tests failed with timeout/empty output; code inspection showed `CreatePseudoConsole` success was inverted.
- Root cause: Win32 `S_OK` is HRESULT zero, but zero entered the fallback branch.
- Change: non-zero HRESULT is now the honest fallback path; zero proceeds through ConPTY.
- GREEN: terminal and host packages cross-compiled for Windows; Linux targeted tests pass. Native Windows execution remains pending CI.

### FC-19 — credential platform boundary

- RED: prior Darwin CI ran Linux Secret Service assertions; the test file only skipped Windows.
- Root cause: D-Bus/Secret Service implementation and tests were in common files.
- Change: credential implementations and Secret Service tests are build-constrained by platform; Darwin/Windows remain truthful unsupported capabilities.
- GREEN: Linux runtime tests pass; Darwin and Windows runtime test binaries cross-compiled.

### FC-22/23 — desktop build proof

- Change: `webkit2_41` is selected only for Linux; CI now has native Wails build lanes for Linux, Windows and macOS, and snapshot depends on all three.
- Local evidence: Linux `go build -o /tmp/nexus-desktop ./cmd/nexus-desktop` passed.
