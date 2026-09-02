# IAPro Nexus — Final Local Validation Report

**Product:** IAPro Nexus V0 canonical rebuild
**Branch:** `feat/nexus-canonical-review-rebuild`
**Functional implementation checkpoint:** `1e6404ac3dc487b78b1439a731a5ad47d16a1933`
**Date:** 2026-09-01

## Executive verdict

**READY_FOR_PLATFORM_VALIDATION**

The canonical local implementation is complete for the requirements that can be implemented and exercised in this environment. This verdict does **not** mean production release approval. Full Go 1.25/backend transport, native Windows/macOS, real-provider, same-SHA CI and release gates remain external/unverified here.

## What was validated locally

- TypeScript compilation
- ESLint
- complete frontend Vitest suite
- production web build
- deterministic Chromium UI bundle E2E
- Mission Runner unit/durability/evidence tests executable with cached dependencies
- Runner `go vet`
- selected pure-Go Nexus package tests/vet
- Go source syntax parsing
- `git diff --check`
- capability-preservation source audit
- canonical-product alignment audit

## Browser E2E result

**PASS** — actual current React/XTerm bundle in Chromium with deterministic authenticated/CSRF fixtures.

Covered journey:

`Project → Composer MISSING → Ask existing Agent → Project Shell → Tabs/Desktop → Context READY → Flow Draft A→B||C→D → Approve & Run → Flow Run → dependency receipts / WorkReceipt`.

The test asserts zero execution side effects while the Flow is still a Draft.

## Backend/auth boundary

The backend contains E2E source coverage for one-time bootstrap, session cookie, CSRF, Origin and WebSocket authentication. This environment cannot compile/run that full package because required Go 1.25 and external modules cannot be downloaded. Therefore backend auth transport is **UNVERIFIED LOCALLY**, not silently promoted to PASS.

## Full Go boundary

**UNVERIFIED_TOOLCHAIN/DEPENDENCIES** in this sandbox:

- `go test ./...`
- `go test -race -count=1 ./...`
- `go vet ./...`
- current `go build ./cmd/nexus`

Reason: Go 1.25 and required external modules are absent; outbound network/DNS is blocked.

## Platform gates still required

1. Same final SHA on Go 1.25 with full module graph: test/race/vet/build.
2. Backend auth/WebSocket E2E against the real local server.
3. Native Windows execution including ConPTY/named-pipe terminal paths.
4. Native macOS execution.
5. Real-provider E2E for supported CLIs/accounts, separate from deterministic fake-provider Core E2E.
6. Same-SHA CI/security checks.
7. Release packaging/signing/publishing if applicable.

## Artifact policy

The final distributable source ZIP intentionally excludes `.git`, worktrees, node_modules, caches, recovery artifacts, credentials/test databases and the tracked stale prebuilt `nexus` binary from an older SHA. The current binary must be built from the final source SHA in the Go 1.25 platform-validation environment.
