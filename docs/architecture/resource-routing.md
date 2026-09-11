# Resource routing

`ResourceScheduler`/`RecommendResources` is the canonical provider-profile
allocator. It applies authentication, availability, cooldown, rate-limit and
required-capability hard gates before scoring quota confidence, health,
continuity, policy and capability fit. UNKNOWN quota is never treated as full
capacity.

Agent matching is intentionally separate: an Agent specializes the work, while
the ResourceScheduler chooses an executable provider resource.

Current evidence: recommendation tests and local Web/Go gates. Full automatic
failover E2E remains pending.
