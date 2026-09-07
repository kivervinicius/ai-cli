# META Plan: Code Review Fixes

## Execution Strategy
4 parallel lanes, each with independent subagents. Lane 1 (Security) runs first because C1 blocks everything. Lanes 2-4 can run in parallel after Lane 1.

## Lane 1: SECURITY (C1 + H7) — CRITICAL PATH
**Agent:** cavecrew-builder (surgical fixes)
**Estimated iterations:** 2

### Tasks
1. **C1: Command injection fix** — `internal/nexus/runner/verifier.go`
   - Add command allowlist validator
   - Parse commands into argv without shell wrapper
   - Reject shell metacharacters
   - Add tests for injection attempts

2. **H7: Directory creation allowlist** — `internal/control/web/handlers_fs.go`
   - Add `isWithinAllowedRoots()` check
   - Restrict to home directory and project roots

### Verification
- `go test ./internal/nexus/runner/... -run TestVerification`
- `go test ./internal/control/web/... -run TestFS`

---

## Lane 2: BACKEND QUALITY (H1-H3, H9, M9-M14) — PARALLEL
**Agent:** cavecrew-builder
**Estimated iterations:** 3

### Wave 2a (independent fixes)
1. **H1: MkdirAll error** — `nexus.go:191` — return error
2. **H2: UpdateAgent error** — `nexus.go:419` — check error, compensate
3. **H3: StopAgent errors** — `nexus.go:457-506` — log or return errors
4. **M9: Body size limit** — `handlers_nexus.go` — add `io.LimitReader`
5. **M10: Deduplicate isQuotaOrRateLimit** — extract to shared package
6. **M14: StopAgent type safety** — change signature to `(ctx, agentID string)`

### Wave 2b (depends on 2a)
7. **H9: Race condition** — `runner.go:600` — add mutex or document invariant
8. **M11: All accounts unavailable** — `app.go:316` — return error
9. **M12: GetCachedUsage refactor** — `quota.go:69` — extract helpers
10. **M13: ExecuteNextStep refactor** — `runner.go:96` — extract state handlers

### Wave 2c (depends on 2b)
11. **L1: Dead code** — `store.go:78` — remove unused `applied`
12. **L2: cloneRun errors** — `repository.go:37` — handle marshal errors
13. **L8: Config error handling** — `app.go` — propagate in main paths

### Verification
- `go test ./internal/... -count=1`
- `go vet ./internal/...`

---

## Lane 3: FRONTEND (H4-H6, H8, M3-M8, M15-M16) — PARALLEL
**Agent:** cavecrew-builder (frontend specialist)
**Estimated iterations:** 3

### Wave 3a (independent fixes)
1. **H4: Error Boundaries** — `web/src/`
   - Create `ErrorBoundary.tsx` component
   - Wrap `<AppRoutes />` top-level
   - Wrap each major surface (workspace, settings, projects)

2. **H6: URL param validation** — `web/src/App.tsx:9`
   - Add zod schema for `WorkspaceSurface`
   - Validate before passing as props

3. **M3: i18next XSS** — `web/src/i18n/index.ts:35`
   - Set `escapeValue: true` or audit all `t()` calls

4. **M8: `as any` escape** — `AgentTerminal.tsx:941`
   - Define proper TypeScript interface for custom element

### Wave 3b (depends on 3a)
5. **H5: TypeScript interfaces** — `web/src/nexus/api.ts`
   - Replace 20+ `any` types with proper interfaces
   - Use existing types from `types.ts`

6. **M4: Auth state encapsulation** — `web/src/api.ts`
   - Move tokens into class with lifecycle management

7. **M5: Consolidate CSRF** — `web/src/api.ts` + `nexus/api.ts`
   - Single source of truth for CSRF token

8. **M6: AbortController** — `PlanBuilderSurface.tsx`
   - Add AbortController to all API-fetching effects

### Wave 3c (depends on 3b)
9. **M7: JSON.stringify perf** — `PlanBuilderSurface.tsx:297`
   - Memoize with hash or incremental dirty tracking

10. **M15: Polling backoff** — `PlanBuilderSurface.tsx:239`
    - Add exponential backoff

11. **M16: Console logging** — `api.ts:94`
    - Guard with `import.meta.env.DEV`

12. **H8: Component split** — `PlanBuilderSurface.tsx` (2105 lines)
    - Extract hooks: `usePlanBuilder`, `useAgentRun`
    - Split into child components

### Verification
- `npm --prefix web run typecheck`
- `npm --prefix web run lint`
- `npm --prefix web run test`
- `npm --prefix web run build`

---

## Lane 4: SECURITY HARDENING (M1-M2, L3, L11) — PARALLEL
**Agent:** cavecrew-builder
**Estimated iterations:** 1

### Tasks
1. **M1: Bootstrap token** — `server.go:285`
   - Implement one-time authorization code flow
   - Exchange via POST for session cookie

2. **M2: CSP hardening** — `server.go:290`
   - Add nonces for inline scripts
   - Restrict `connect-src` to known origins

3. **L3: LastKnownTTL** — `quota.go:21`
   - Reduce from 3650 days to 30 days

4. **L11: Test binaries** — `.gitignore`
   - Add `*.test` to gitignore
   - Remove tracked test binaries

### Verification
- `go test ./internal/core/quota/... -count=1`
- Manual CSP header inspection

---

## Execution Order

```
Time ──────────────────────────────────────────────────►

Lane 1: [C1+H7 fix] ──[verify]──►
                                    │
Lane 2: ───────[Wave 2a]──[Wave 2b]──[Wave 2c]──[verify]──►
                                                          │
Lane 3: ───────[Wave 3a]──[Wave 3b]──[Wave 3c]──[verify]──►
                                                          │
Lane 4: ───────[fixes]──[verify]──►                      │
                                                          │
                              ┌───────────────────────────┘
                              ▼
                    [FINAL VERIFICATION]
                    - go test ./internal/...
                    - npm test
                    - npm build
                    - Architect review
```

## Final Verification Checklist
- [ ] All CRITICAL/HIGH findings fixed
- [ ] All MEDIUM findings fixed where safe
- [ ] `go test ./internal/...` passes (625+ tests)
- [ ] `go vet ./internal/...` clean
- [ ] `npm --prefix web run typecheck` passes
- [ ] `npm --prefix web run lint` passes
- [ ] `npm --prefix web run test` passes
- [ ] `npm --prefix web run build` succeeds
- [ ] Architect review approved
