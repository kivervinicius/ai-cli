# IAPro Nexus V1 — Resource Recommendation & Scheduler Report

## 1. Overview
The Resource Recommendation Service (`internal/nexus/resource_recommendation.go`) provides explainable scoring and provider allocation based on real-time quotas, health status, task capabilities, and scheduling policies.

## 2. Multi-Criteria Scoring Model
- **Quota & Capacity (30%)**: Normalized quota remaining with honest confidence levels (`LIVE`, `CACHED`, `UNKNOWN`). Unknown quota is never falsely scored as best.
- **Health & Reliability (20%)**: Live process checks and degraded state penalties.
- **Affinity & Continuity (20%)**: Favors active session reuse to prevent unnecessary context switching.
- **Role & Capability Fit (20%)**: Matches reviewer, implementer, architect, and tester roles to provider capabilities.
- **Policy Enforcement (10%)**:
  - `BALANCED`: Default weighted compromise.
  - `PRESERVE_QUOTA`: Favors accounts with >70% quota and penalizes accounts <30%.
  - `PREFER_PROVIDER`: Enforces explicit provider/profile preferences.
  - `MANUAL`: Direct user override.

## 3. Verification Evidence
- Unit tests: `internal/nexus/resource_recommendation_test.go` (100% pass under `-race`).
- REST endpoint: `POST /api/v1/resources/recommend` wired and validated.
