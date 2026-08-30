# IAPro Nexus — Platform Matrix

| Capability | Linux | macOS | Windows | Evidence in this package |
|---|---|---|---|---|
| Path/DataDir abstraction | Implemented | Implemented | Implemented | `filepath`, config DataDir abstractions |
| Terminal backend | PTY | PTY | ConPTY | platform-specific terminal files/tests |
| Control transport | Unix socket | Unix socket | Named Pipe | platform-specific protocol endpoints |
| Process lifecycle | implemented | implemented | implemented | platform-specific launcher/terminal code; CI jobs |
| Browser open | xdg/browser | `open` | platform browser | runtime/browser abstractions |
| Installer | `install.sh` | `install.sh` | `install.ps1` | scripts present |
| Go build/test CI | Go 1.25 | Go 1.25 | Go 1.25 | `.github/workflows/ci.yml` |
| Race detector CI | enabled | enabled | normal tests | CI configuration |
| Release archive | tar.gz | tar.gz | zip | `.goreleaser.yaml` |
| Physical runtime validation in this sandbox | partial Linux only | **not run** | **not run** | release condition |

## CI matrix

The final workflow contains dedicated jobs for:

- Ubuntu: gofmt, vet, race suite, PTY/Unix socket, runtime/web E2E, binary build.
- Windows: vet, test, ConPTY, Named Pipe, runtime/web E2E, `nexus.exe`, PowerShell smoke.
- macOS: vet, race suite, PTY/socket/runtime/web E2E, binary build, installer syntax smoke.
- GoReleaser snapshot depends on all three OS jobs plus frontend.

## Static evidence obtained in sandbox

- Linux and Darwin `internal/control/protocol` and `internal/control/terminal` compile/test paths were exercised with local dependency stubs.
- Windows protocol cross-compile reached the Windows-specific `go-winio` dependency but could not download it because the sandbox has no network/cache entry.
- Full binary cross-build was similarly blocked by uncached Go modules, not by a source compiler diagnostic.

**Rule:** do not change this matrix to “validated” until the final commit's GitHub Actions runs pass on real hosted OS runners.
