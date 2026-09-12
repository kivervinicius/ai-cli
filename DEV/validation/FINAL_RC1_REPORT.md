# FINAL RC1 REPORT (WIP)

STATUS: CURRENT  
Starting SHA: `3985908244cc01c9e08c5ad4da0854622673a3d7`  
Final SHA (so far): see `git rev-parse HEAD` on `fix/nexus-rc1-closure`  
Branch: `fix/nexus-rc1-closure`

## RC-SECURITY

| Item | Status |
|------|--------|
| EffectiveExposure | PASS (local tests) |
| Tunnel → PUBLIC_REMOTE | PASS |
| Remote bootstrap one-time + TTL | PASS |
| Secure cookies under tunnel | PASS |
| Remote sessions not restored | PASS |
| cloudflared pin + SHA256 | PASS (checksums refreshed 2026-09-12) |
| PATH cloudflared | PARTIAL (accepted only if version matches pin) |

## RC-RELEASE (partial)

| Item | Status |
|------|--------|
| Embedded Ed25519 trust root | PASS (public key embedded) |
| install.sh/ps1 signed manifest required | PASS (local contract tests) |
| Private key in GitHub Actions | BLOCKED_EXTERNAL |
| Desktop artifacts in signed manifest | PENDING |
| Same-SHA multiplatform CI | PENDING (historical CI red on older SHAs) |

## Remaining for GO

1. RC-STABILITY: Windows/Browser/macOS desktop gates on real runners
2. Wire release CI secret `NEXUS_SIGNING_PRIVATE_KEY` matching embedded pubkey
3. Consecutive green CI on this SHA
4. Independent security review of tunnel+installer

## Verdict (interim)

**NO_GO** — RC-SECURITY local slice landed; full RC Definition of Done not yet met.
