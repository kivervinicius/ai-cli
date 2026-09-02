# Nexus V0 Local Execution Report

**Branch:** `feat/nexus-canonical-review-rebuild`
**Functional implementation checkpoint:** `1e6404ac3dc487b78b1439a731a5ad47d16a1933`
**Date:** 2026-09-01

## Implemented local product path

The current source implements the canonical path:

`Project → Agent/Shell/Composer → Context READY → Flow Draft → Approve & Run → durable Mission Runner → ContextCapsule → provider execution → verification/review → WorkReceipt → dependency receipt handoff → Flow Run`.

The deterministic Flow contract `A → B || C → D` is covered at model/runner level and in the browser UI fixture.

## Fresh local verification evidence

### Frontend

Executed on the current source checkpoint during the finalization cycle:

- TypeScript typecheck: PASS
- ESLint: PASS
- Vitest: PASS — 24 files / 90 tests
- Production web build + embedded control-web bundle: PASS

### Deterministic Chromium bundle E2E

Command:

`python3 scripts/nexus_browser_e2e.py --artifacts <dir>`

Result: PASS.

Assertions exercised through the actual React/XTerm bundle:

1. Composer shows Context MISSING and blocks Turn into Flow.
2. Ask targets the existing Agent without creating another Agent.
3. Project Shell starts independently, emits terminal fixture output, and closing it stops only that shell runtime.
4. Tabs → Desktop → Tabs preserves the logical Composer view.
5. Context preparation reaches READY.
6. Flow Draft displays deterministic Wave 1 / Wave 2 / Wave 3 for A → B || C → D.
7. Draft authoring causes no `/run` mutation.
8. Approve & Run creates the run only after explicit approval.
9. Flow Run reaches the deterministic completed fixture.
10. D shows two dependency receipt inputs and factual WorkReceipt content.
11. All mocked write requests carry the session CSRF token; zero CSRF failures.

Because this sandbox's Chromium policy blocks all URL navigation (including loopback and file://), the web bundle is injected directly into a real Chromium page and uses deterministic browser-native fetch/WebSocket/localStorage fixtures. This validates the current browser UI bundle and mutation semantics, but does not substitute for backend cookie/Origin transport E2E.

### Go / Runner

Executable locally using a temporary, non-committed `go.mod` directive downgrade solely for packages whose dependencies are already cached:

- `go test ./internal/nexus/runner`: PASS
- `go vet ./internal/nexus/runner`: PASS
- tests/vet for autonomyguard, contextsnapshot, intelligence, maestrogates: PASS
- Go source parser over current source: PASS

The repository `go.mod` remains at the required Go version; the temporary downgrade is never committed.

## Not locally executable in this sandbox

Full `go test ./...`, race suite, full `go vet ./...` and current Nexus binary build cannot be completed because:

- project requires Go 1.25 while sandbox provides Go 1.23.2;
- Go 1.25 cannot be downloaded because outbound network/DNS is blocked;
- module cache lacks `modernc.org/sqlite`, `github.com/creack/pty` and `github.com/gorilla/websocket`;
- no vendored module tree is present.

These gates are **UNVERIFIED_TOOLCHAIN/DEPENDENCIES**, not PASS and not a discovered source failure.

## Backend auth E2E

`internal/control/web/e2e_test.go` contains the backend transport test for one-time bootstrap, authenticated HttpOnly session, CSRF, Origin validation and terminal WebSocket authorization. It is not claimed PASS locally because the full Go dependency graph cannot be compiled in this sandbox.
