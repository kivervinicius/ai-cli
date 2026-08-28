# Nexus V1 — Acceptance, UX Validation & Release Checklist

## Acceptance (charter §121-147)

| ID | Requirement | Status this gate |
|---|---|---|
| A1 | Project persists across restart, same ID | ✅ live-verified (project survived web restart, same `prj_…`) |
| A2 | Agent persists; same AgentID across runtime | ✅ store+API (agent id stable); browser-close continuity is Gate 4 |
| A3 | Output while disconnected → bounded replay | design holds (ring buffer); Gate 4 test |
| A4-A7 | LIVE/RESTART config + honest continuity | Gate 3 (ImpactAnalyzer/Reconfigure) |
| A8-A9 | Rate-limited → Account/Context handoff recommendation | Gate 5 scheduler |
| A10-A12 | Multi-project, two browsers, web restart | Gate 4/5; store isolation tested now |
| A13 | Machine reboot → RECOVERABLE agent | Gate 4/8 (recover flow) |
| A14-A16 | Maestro ASSIST / OFF / unavailable | Gate 6 |
| A17 | Mission Beta | Gate 7 (salvage) |
| A18-A20 | Path security / Origin / IDOR | ✅ implemented + tested |
| A21-A27 | Slow viewer / large output / attach races / resize / reconfigure failure / handoff failure / redaction | prior gate + ongoing |

## UX validation (this gate)

- Web shell: nav (Nexus + Legacy), command palette (Ctrl/Cmd+K), project rail,
  agent cards with status badges, empty states (No Projects / No Agents), agent
  terminal frame. All state real API — no fixtures in production.
- Loading (`Spinner`), error (inline), empty (`EmptyState`), stale (disconnected
  terminal message) states present.
- Responsive: desktop-first; the shell degrades to the nav rail + content on narrow
  widths. Mobile minimum-operational is a Gate 8 item.
- Visual QA screenshots (1440/1280/390) are a Gate 8 item.

## Release checklist (RC path, §181)

- [x] Baseline validated
- [x] SQLite store + migrations + tests
- [x] Project model + API
- [x] Persistent Agent model + API + start/stop
- [x] Agent terminal WS (101 verified)
- [x] Web-first shell + tokens + primitives
- [x] Full `-race` suite (25 pkgs) + vet + frontend build
- [ ] Gate 2: agent detail/lineage UI, recover flow
- [ ] Gate 3: configuration drawer + safe apply (restart/resume/verify)
- [ ] Gate 4: terminal broker stability, browser close/reopen, layout restore
- [ ] Gate 5: Resource Scheduler + policies
- [ ] Gate 6: Maestro bridge + Assist UI
- [ ] Gate 7: Mission Beta (flagged off default)
- [ ] Gate 8: a11y, visual QA, installers, `iapro` CLI + doctor, Windows/macOS local E2E
- [ ] Release: `v1.0.0-rc.1`, GoReleaser snapshot, artifact matrix + checksums, **no publish without approval**

## RC verdict rule (§182-185)

`GO` only if all gates + independent final review + Windows/macOS runtime evidence
pass. `CONDITIONAL_GO` if the only limitation is clearly isolated (e.g. Mission Beta
off by default). `NO_GO` for terminal cross-talk, agent state loss, browser/web
restart killing the provider, unsafe exposure, secret persistence, false verified
resume, writer race, migration data loss, or unproven platform claims.
