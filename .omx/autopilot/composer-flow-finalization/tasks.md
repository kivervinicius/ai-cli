# Task lanes

- [P] Composer domain: archetype-aware brief, unknown/question engine, readiness, external prompt import, skill application, prompt versions. Owner: Composer backend/frontend files. Verify: Composer unit/API tests and frontend tests.
- [P] Flow domain: reusable definition/revision/library, visual graph editing, step/leader/agent inspector, preflight, materialization and snapshot semantics. Owner: Flow/store/API/frontend files. Verify: DAG/revision/snapshot tests and frontend tests.
- [P] Runtime/terminal domain: scheduling, live Flow execution, transport/co-attach/backpressure and cross-platform contracts. Owner: runner/launcher/terminal files. Verify: focused runtime tests, race tests where supported.
- [P] UX/QA domain: design tokens, responsive/accessibility states, E2E/visual regression and final reports. Owner: web UX/tests and DEV reports. Verify: `make web-verify` plus report artifacts.
- Integration gate: lead reconciles APIs and runs full verification before completion.
