# AI CLI — Final Provider Matrix

**Date:** 2026-08-28 · **Branch:** `fix/control-production-readiness`

Derived from `EffectiveCapabilities` (truthful: ProviderSupports ∧ AIControlImplements
∧ PlatformSupports ∧ VersionCompatible ∧ RuntimeProbePasses). Statuses: SUPPORTED /
PARTIAL / UNSUPPORTED / UNKNOWN / NOT_TESTED.

## Codex

| Capability | Status | Note |
|---|---|---|
| Process / Terminal | SUPPORTED | TERMINAL mode; backend = `terminal.BackendMechanism()` |
| Attach | SUPPORTED | IPC socket/pipe |
| StructuredEvents | UNSUPPORTED | no `codex app-server` adapter (honest) |
| SubmitPrompt | UNSUPPORTED | requires app-server adapter |
| Approvals | UNSUPPORTED | no programmatic Approve/Reject |
| Resume | SUPPORTED* | `codex resume <id>`; *not runtime-verified against live provider |
| CancelTurn | SUPPORTED | Ctrl+C passthrough (VT input on PTY/ConPTY) |
| ControlLevel | TERMINAL | truthful downgrade |

## OpenCode

| Capability | Status | Note |
|---|---|---|
| Process / Terminal | SUPPORTED | TERMINAL mode |
| Attach | SUPPORTED | IPC socket/pipe |
| StructuredEvents | UNSUPPORTED | no `opencode serve` adapter (honest) |
| SubmitPrompt | SUPPORTED | `opencode run` |
| Approvals | UNSUPPORTED | no structured approve/reject |
| Resume | SUPPORTED* | `opencode -s <id>`; *not runtime-verified against live provider |
| Fork / Headless / NativeUIAttach | SUPPORTED | signature-level |
| ControlLevel | TERMINAL | truthful downgrade |

## Claude Code

| Capability | Status | Note |
|---|---|---|
| Process / Terminal / Attach | SUPPORTED | TERMINAL mode |
| Resume | SUPPORTED* | `claude --resume <id>` |
| ControlLevel | TERMINAL | |

## Gemini CLI

| Capability | Status | Note |
|---|---|---|
| Process / Terminal / Attach | SUPPORTED | TERMINAL mode |
| Resume | SUPPORTED* | session-based |
| ControlLevel | TERMINAL | |

## AGY / Antigravity

| Capability | Status | Note |
|---|---|---|
| Process / Terminal / Attach | SUPPORTED | TERMINAL mode |
| Resume | SUPPORTED* | `agy --conversation=<id>` |
| Isolation | SUPPORTED | isolated D-Bus / keyring (credential isolation, not sandbox) |
| ControlLevel | TERMINAL | |

## Cursor Agent

| Capability | Status | Note |
|---|---|---|
| Detection / TERMINAL | SUPPORTED | multi-path auto-discovery |

## Detection truth

- `Detect()` gates `Installed` on real binary lookup + `--version` output; unexecutable
  binaries are marked uninstalled. Capabilities never claim SUPPORTED for an
  uninstalled provider (runtime probe required).
- Approvals are only SUPPORTED when AI Control can structurally `Approve()/Reject()`;
  a terminal `Allow?` prompt alone is not treated as programmatic approval support.

## Fake test provider

`fake` is registered for E2E/structural tests only (hidden from user-facing lists) and
drives the interactive-terminal, session, resume, and failure-mode test matrix.
