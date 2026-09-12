# RC1 Closure Ledger

- Branch: `fix/nexus-rc1-closure`
- Worktree: `.worktrees/nexus-rc1-closure`
- Starting SHA: `3985908244cc01c9e08c5ad4da0854622673a3d7`
- Feature freeze: ON
- Phase: RC-SECURITY

## Status

| Milestone | Status |
|-----------|--------|
| RC-SECURITY | IN_PROGRESS |
| RC-STABILITY | PENDING |
| RC-RELEASE | PENDING |
| RC-GOVERNANCE | PENDING |
| FINAL-RC-VERIFICATION | PENDING |

## Discover notes

- Tunnel already has `tunnelActive` → one-time bootstrap + Secure cookies.
- cloudflared version pinned `2026.8.2` with SHA verify on managed download.
- Gaps: explicit EffectiveExposure API, remote bootstrap TTL, no persist under PUBLIC_REMOTE, darwin checksums look placeholder, PATH cloudflared unverified.
