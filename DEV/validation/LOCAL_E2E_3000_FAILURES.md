# IAPro Nexus — E2E failures and limitations

## E2E-001 — canonical port occupied

- Severity: medium, environmental.
- Root cause: pre-existing PID 385021 owns `127.0.0.1:3000`.
- Correction: none; it was not started by this run and was preserved.
- Retest: startup/bootstrap/cleanup passed on port 3001.

## E2E-002 — no authenticated real provider

- Severity: external prerequisite.
- Root cause: resource selection correctly rejects profiles without real authentication.
- Correction: none; fabricated authentication is forbidden.
- Retest: harness reports `NEXUS_E2E_NOT_AUTHENTICATED` (exit classification 2).

## E2E-003 — browser executable unavailable

- Severity: low, evidence gap.
- Root cause: Playwright CLI is installed, but its managed Chromium executable is not downloaded in the environment.
- Correction: browser smoke now reports `SKIP`/exit classification 2 rather than claiming PASS or failing the local HTTP gate.
- Retest: startup with `--browser` reaches this explicit classification and still performs Nexus cleanup.
