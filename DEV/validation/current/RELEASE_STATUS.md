# Current release status

STATUS: CURRENT  
GIT_SHA: (see `git rev-parse HEAD` at generation; do not copy stale SHA from historical FINAL_* reports)  
GENERATED_AT: 2026-09-12  
ENVIRONMENT: linux/amd64 go1.25.0  
EVIDENCE_STREAM_ID: see hardening report

This is the **only** document that should be treated as the current release
status for HEAD. Files named `FINAL_*` under `DEV/validation/` without this
directory are historical projections unless they also carry STATUS: CURRENT
and the same SHA.

Canonical evidence is `ValidationEvidenceStream`, not Markdown PASS claims.

Local hardening (2026-09-12): `make quality` PASS, `make attention-e2e` PASS,
`make overnight-smoke` PASS. Overnight 8h soak, live providers, native
Windows/macOS remain UNVERIFIED. Details:
`DEV/validation/current/FINAL_HARDENING_REPORT.md`.

## How to refresh

```bash
make quality
make attention-e2e
make overnight-smoke
git rev-parse HEAD
```

Then update `DEV/validation/current/FINAL_HARDENING_REPORT.md`.
