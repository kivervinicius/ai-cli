# Ralph execution context — Nexus terminal continuity

## Task

Execute `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md` from P0 through P12,
covering CI, honest quota observations, grouped resource pools, terminal
supervision/autocomplete, Maestro continuity, transactional handoff and
configurable fallback on Linux, macOS and Windows.

## Desired outcome

A reviewable implementation with fresh verification evidence. Native platform
execution remains mandatory; cross-compilation is supporting evidence only.
No commit, push, merge, tag or remote branch protection change is authorized.

## Known facts

- Worktree is intentionally dirty with extensive pre-existing staged and unstaged changes.
- HEAD is `1899ca6334576d859056d48a394e51d03758f313`.
- Direct `nexus <provider>` currently calls the provider adapter's interactive
  runner; SessionHost slash routing exists only for supervised runtimes.
- Existing CI and local docs contain fixes for the prior SHA, but native macOS
  and Windows evidence must be freshly produced on their runners.
- Quota data must remain UNKNOWN/DEGRADED when no verified source exists.

## Constraints

- Preserve unrelated user changes; use apply_patch for edits.
- Follow AGENTS, Maestro and DEV contracts.
- Do not make quota numbers or continuity claims without source/evidence.
- Keep Maestro optional; never install it silently.
- New frontend styles use SCSS Modules, tokens and i18n.

## Open risks

- Native Windows ConPTY/Named Pipe and macOS PTY behavior cannot be proved on this Linux host.
- Multi-process quota monitoring needs a real lease/fencing design, not an in-process mutex.
- Cross-provider handoff cannot claim provider-native conversation memory.
- Full P0–P12 scope is large; each completed package must be documented and gated.

## Likely touchpoints

`internal/core/quota`, `internal/profile`, `internal/nexus/quota_monitor*`,
`internal/core/scheduler`, `internal/core/config`,
`internal/control/{launcher,host,protocol,handoff,terminal,registry}`,
`internal/app`, `internal/tui`, `web/src/nexus`,
`.github/workflows`, `DEV/`, `.omx/`.
