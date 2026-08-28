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

Designed; implementation is Gate 5 (not yet built). The existing intra-provider
selector (`internal/core/scheduler`) remains the live capability until then.
