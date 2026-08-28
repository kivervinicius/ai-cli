# Nexus V1 — Platform Matrix

| Platform | Cross-build | vet | unit/race | runtime E2E (CI) | local runtime evidence | Verdict |
|---|---|---|---|---|---|---|
| Linux amd64 | ✅ | ✅ | ✅ 25 pkgs `-race` | ✅ (prior gate) | ✅ (this gate) | **SUPPORTED** |
| Linux arm64 | ✅ | ✅ | ✅ | ✅ | build-only | SUPPORTED (build) |
| Windows amd64 | ✅ | ✅ | ✅ (unit) | ✅ authored (ConPTY/NamedPipe/PS) | pending user | CODE-COMPLETE, runtime pending |
| Windows arm64 | ✅ | ✅ | ✅ (unit) | ✅ authored | pending | CODE-COMPLETE, runtime pending |
| macOS amd64 | ✅ | ✅ | ✅ (unit) | ✅ authored (PTY/socket/Web) | pending user | CODE-COMPLETE, runtime pending |
| macOS arm64 | ✅ | ✅ | ✅ (unit) | ✅ authored | pending (Apple Silicon mandatory) | CODE-COMPLETE, runtime pending |

## This-gate live evidence (Linux)

- `ai control web` serves Nexus UI on loopback; project/agent API works end-to-end.
- Agent start → persistent `__control-host` runtime (`fake-…` ULID) RUNNING.
- Agent terminal WS returns **101 Switching Protocols**.
- Project persists across web server restart with the same `prj_…` ID.

## Rule (§148, §156)

Cross-compile is never treated as runtime E2E. Windows/macOS reach SUPPORTED only
after CI (windows/macos runners) and user-local confirmation produce runtime evidence.
The RC verdict depends on this (§185).
