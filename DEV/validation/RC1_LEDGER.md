# RC1 Closure Ledger

- Branch: `fix/nexus-rc1-closure`
- Worktree: `.worktrees/nexus-rc1-closure`
- Starting SHA: `3985908244cc01c9e08c5ad4da0854622673a3d7`
- Feature freeze: ON
- Phase: RC-STABILITY / RC-RELEASE (partial)

## Status

| Milestone | Status |
|-----------|--------|
| RC-SECURITY | PASS (local) — EffectiveExposure, tunnel PUBLIC_REMOTE, remote bootstrap TTL, Secure cookies, cloudflared pin+SHA, signed install trust root |
| RC-STABILITY | IN_PROGRESS — /tmp TempDir, SessionHost readiness, Close() errors, version contract, E2E storage reset, macOS plist version assert |
| RC-RELEASE | PARTIAL — signed manifest required in installers; CI signing secret + desktop in manifest still BLOCKED_EXTERNAL / PENDING |
| RC-GOVERNANCE | PENDING |
| FINAL-RC-VERIFICATION | WIP report exists — interim **NO_GO** |

## Notes

- Private signing key under `DEV/validation/rc1-secrets/` is gitignored; must be GitHub Actions secret.
- Native Windows/macOS evidence still required for GO.
