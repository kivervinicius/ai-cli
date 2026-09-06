# IAPro Nexus — Local E2E report

Date: 2026-08-30  
Branch: `feat/nexus-maximum-delivery`

The versioned harness is [`scripts/nexus-e2e-local.go`](../../scripts/nexus-e2e-local.go). With `--start`, it creates a temporary Git project and isolated `NEXUS_DATA_DIR`, starts Nexus with `--no-open`, consumes the bootstrap URL in memory, and stops/removes process and data. Tokens are not printed.

| Check | Result | Evidence |
|---|---|---|
| Port 3000 | PASS | Canonical startup rerun completed; no listener remained from this run. |
| Local startup/HTTP bootstrap | PASS | Harness on ports 3000/3001; isolated Git/data and cleanup. |
| Provider discovery | PASS | Existing Codex profile `kivergmail` was imported into isolated data and selected explicitly. |
| Real runtime/WS/provider output | PASS | Compiled harness exited 0 with exact-line `NEXUS_E2E_OK` marker, runtime identity, CONTROL and generation checks. |
| Browser CLI | AVAILABLE | `npx --yes playwright --version` → `10.9.2`; Chromium installed outside the repository. |
| Browser smoke | PASS | Token-safe smoke completed with exit 0; no token copied to artifacts. |

When authentication is available, the harness covers bootstrap/auth, project, persistent Agent, manual provider/profile allocation, runtime identity, Agent-scoped WebSocket, CONTROL lease, prompt/output, RuntimeGeneration persistence, stop/cleanup, and Agent identity after refresh/reconnect.

Context handoff, Safe Apply generation restart, Mission real execution, remediation, restart durability, and Take Control were not executed by this harness and remain `NOT_TESTED`.

No production secrets, cookies, bootstrap tokens, prompts, or provider output were written to reports. Local Direct Work verdict: `PASS`; overall production delivery verdict: `BLOCKED` because required Mission/remote CI/fresh-clone evidence is unavailable.
