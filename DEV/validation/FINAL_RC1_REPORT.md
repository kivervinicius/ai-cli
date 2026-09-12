# FINAL RC1 REPORT (WIP)

STATUS: CURRENT  
Starting SHA: `3985908244cc01c9e08c5ad4da0854622673a3d7`  
Branch: `fix/nexus-rc1-closure`  
Final SHA: see `git rev-parse HEAD`

## RC-SECURITY

| Item | Status |
|------|--------|
| EffectiveExposure | PASS (local) |
| Tunnel → PUBLIC_REMOTE | PASS |
| Remote bootstrap one-time + TTL | PASS |
| Secure cookies under tunnel | PASS |
| Remote sessions not restored | PASS |
| cloudflared pin + SHA256 | PASS |
| Embedded Ed25519 trust root | PASS |
| Private signing key in CI | BLOCKED_EXTERNAL |

## RC-STABILITY

| Item | Status |
|------|--------|
| E2E `/tmp` → TempDir | PASS (local) |
| SessionHost QA readiness (no fixed sleep) | PASS (local) |
| SQLite Close() error checks on reopen | PASS (local) |
| `/tmp` allowlist gated non-Windows | PASS (local) |
| VERSION ↔ package.json ↔ wails.json in `make quality` | PASS |
| E2E localStorage reset + waitForFunction | PASS (code); browser CI PENDING |
| macOS CFBundleShortVersionString CI assert | PASS (workflow); native run PENDING |
| Native Windows/macOS runners green on this SHA | PENDING |

## RC-RELEASE

| Item | Status |
|------|--------|
| install.sh/ps1 require signed update-manifest | PASS (local) |
| Desktop artifacts in signed manifest | PENDING |
| Authenticode / notarization | BLOCKED_EXTERNAL |

## Verdict (interim)

**NO_GO** — security + stability slices landed locally; multi-platform CI + signing secret + desktop manifest still open.
