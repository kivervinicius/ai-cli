# Nexus V1 — Resource Scheduler (design; Gate 5)

Charter §42-48, §128. Evolve the intra-provider selector into a cross-provider
**Resource Scheduler** deciding provider + profile + account.

## Model

```
Resources = Provider → Profiles/Accounts { health, availability, freshness, confidence, cooldown, capabilities }
UsageSnapshot { provider, profile, status, windows[], source, freshness, confidence }
Status: LIVE | CACHED | ESTIMATED | UNKNOWN | RATE_LIMITED | UNAVAILABLE   // UNKNOWN ≠ 100%
```

## Scheduler inputs (§45)

AgentPolicy · TaskRequirements · ProviderCapabilities · Availability · Health ·
Continuity · ProjectPolicy · UserPreference · Cooldown · RateLimitRisk

## Explainable score (§46)

```
score = capabilityMatch + normalizedAvailability + health + continuity
      + projectAffinity + userPreference − cooldown − rateLimitRisk − switchingCost
```

## Allocation policies (§47)

`Balanced` · `Preserve Quota` · `Prefer Provider` · `Manual`.
Cost-optimized explicitly deferred.

## Policy hierarchy (§48)

`Global → Project → Agent` (preferred/forbidden providers, min availability,
fallback order, auto account handoff).

## Outputs

- Allocation **explanation** (why this provider/profile/account).
- Recommendations surfaced as actions: `recommend Account Handoff` (same provider,
  rate-limited account, §128) and `recommend Context Handoff` (provider exhausted,
  NEW SESSION, §129). Auto only per policy.
- Same reusable ResourcePicker component for Start Agent / Reconfigure / Account
  Handoff / Mission assignment (§176).

## Status

**Implemented.** Resource discovery, allocation, and explainable selection are live.

### Backend (`internal/nexus/resource_discovery.go`)
- `ListResources()` — discovers provider accounts via `profile.List()` + `driver.Detect()`, returns real health/quota status.
- `AllocateResource()` — validates an exact provider/profile choice, persists it as the Agent's `AgentConfig` revision.
- `ResolveStartParams()` — resolves provider/profile: explicit params > AgentConfig > `REQUIRED_RESOURCE_SELECTION` error.
- `validateResourceAccount()` — rejects unauthenticated, rate-limited, cooldown, or unavailable resources.

### API
- `GET /api/v1/resources` — returns real discovered accounts.
- `POST /api/v1/resources/select` — validates and persists selection to agent config.

### Frontend
- `ResourcePicker.tsx` — consumes `listResources()` + `selectResource()`, shows quota bars, health badges, availability status.
- `AgentsSurface.tsx` — start button triggers `REQUIRED_RESOURCE_SELECTION` → opens `ResourcePicker` modal.

### Flow
```
1. UI calls startAgent(agent.id) — no provider/profile
2. Backend returns 409 REQUIRED_RESOURCE_SELECTION
3. UI opens ResourcePicker → GET /api/v1/resources (real discovery)
4. User selects → POST /api/v1/resources/select (persists to AgentConfig)
5. UI calls startAgent(agent.id) again
6. Backend reads AgentConfig → resolves provider/profile → launches
```

### Remaining (future gates)
- Cross-provider scoring with continuity, cooldown, and rate-limit risk.
- Project-level and global policy hierarchy (preferred/forbidden providers).
- Account Handoff and Context Handoff recommendations.
