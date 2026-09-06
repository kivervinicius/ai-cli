# Capacity Monitor Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an opt-in, single-leader Nexus capacity monitor that observes every enabled provider profile, emits honest quota state transitions and explainable recommendations, delivers canonical in-app/browser/native alerts, and provides testable Linux background startup.

**Architecture:** Read-only provider probes and a privacy-safe process detector feed one adaptive monitor in `internal/nexus/capacitymonitor`. The monitor persists observations, leases, state transitions, fingerprints, preferences, and delivery receipts in the existing SQLite store, publishes canonical events through `internal/control/events`, and delegates candidate ranking to Nexus resource recommendation code. The authenticated Web API and existing notification center/settings render backend state; Linux autostart is a separate opt-in adapter, while desktop tray selection and remote push remain outside this delivery.

**Tech Stack:** Go 1.25, standard library, existing provider adapters and `model.UsageSnapshot`, SQLite through `modernc.org/sqlite`, existing events bus and `internal/control/notify`, authenticated `net/http` API, React 19 + TypeScript 5.9 strict, Vitest, React Testing Library, SCSS Modules, CSS custom properties, `react-i18next`, Linux XDG/systemd user service.

**Spec:** `docs/superpowers/specs/2026-09-05-capacity-monitor-notifications-design.md`

## Global Constraints

- Monitor every enabled `provider + profile + quota group/window`, including profiles not attached to active Agents.
- Read-only probes may use official provider APIs, official read-only CLI commands, provider-owned local files, and Nexus rate-limit evidence; probes that can authenticate, refresh, or mutate configuration are ineligible for background collection.
- External activity may inspect only executable names and resolved executable paths; it reports provider activity, never account, prompt, command, environment, terminal, file, or session attribution.
- One local process owns the monitor lease; collection is capped at two globally and one per provider/profile.
- Intervals are 10m+jitter when healthy/idle, 60s+jitter when active, 30–60s for warning/critical subject to probe minimums, at least 5m in battery saver, and backoff 1/2/5/15m for offline or repeated failure; suspended machines are not woken.
- States are `UNKNOWN`, `HEALTHY`, `WARNING`, `CRITICAL`, `EXHAUSTED`, and `RECOVERED`; defaults are warning 20%, critical 10%, exhausted 0%, recovery 25%, validated by `0 <= exhausted < critical < warning < recovery <= 100`.
- Unknown or stale data never becomes zero capacity or a false exhaustion alert; exact model names are emitted only when attributable evidence supplies them.
- Alert fingerprints are based on provider, profile, group, window, severity, and reset cycle; they exclude email, prompt content, filesystem paths, and tokens.
- All user-visible Web text is localized; use semantic HTML, keyboard focus, SCSS Modules, semantic `--nx-*` tokens, strict TypeScript, and `asArray()` for nullable API collections.
- Background startup is disabled by default and enabled only after explicit consent; exhausted alerts bypass quiet hours only with explicit opt-in.
- Remote mobile push is a non-goal for this plan. Desktop tray is a future increment blocked until a separate desktop-framework decision is approved.
- Follow TDD for each behavior: write one failing test, run it and observe the expected failure, implement the minimum, rerun focused and regression tests, then commit with a Conventional Commit scope.

## File Map

Implementation is expected to touch the following bounded units; this plan itself is the only file being created in the current task.

- Create `internal/core/quota/probe.go` for the dependency-free probe contract and `internal/profile/capacity_probes.go` for provider-specific adapters; this placement prevents the existing `profile -> quota` dependency from becoming an import cycle.
- Modify `internal/core/model/types.go`, `internal/core/quota/view_builder.go`, `internal/profile/usage.go` and tests: source/granularity honesty and the unknown-`ModelName` fix.
- Create `internal/nexus/capacitymonitor/{types.go,state.go,scheduler.go,monitor.go,activity_linux.go,activity_windows.go,activity_darwin.go,power_linux.go,...}`: monitor domain, scheduling, process/power abstraction, and lease orchestration. Platform files use mutually exclusive build tags.
- Create `internal/nexus/store/capacity.go` and `migrations/0013_capacity_monitor.sql` plus tests: additive persistence and idempotent migration.
- Modify `internal/nexus/resource_recommendation.go`, `resource_discovery.go` and tests: monitor input and explainable eligible ranking.
- Modify `internal/control/events/events.go`, `internal/control/notify/notify.go` and tests: canonical capacity event types and channel delivery.
- Modify `internal/control/web/server.go`, `handlers_nexus.go` and add `capacity_monitor_test.go`: authenticated status/history/preferences/acknowledge/snooze/switch endpoints.
- Modify `web/src/nexus/api.ts`, `web/src/types.ts`, `web/src/notifications/*`, `web/src/features/settings/SettingsSurface.tsx`; create colocated `CapacityMonitorPanel.tsx`, `CapacityMonitorPanel.module.scss`, and tests: notification center and Settings controls.
- Create `internal/nexus/capacitymonitor/autostart_linux.go`, `autostart_linux_test.go`, and update the Linux service/install path only where the existing launcher owns lifecycle: consent-gated, testable XDG/systemd user startup.
- Add integration/load tests under `internal/nexus/capacitymonitor` and `internal/control/web`; add release evidence/checklist only in the implementation PR, not as unrelated fixes.

### Task 1: Establish honest quota observation and probe contracts

**Files:**
- Create: `internal/core/quota/probe.go`, `internal/core/quota/probe_test.go`
- Modify: `internal/core/model/types.go`, `internal/core/quota/view_builder.go`, `internal/profile/usage.go`
- Test: `internal/core/quota/view_builder_test.go`, `internal/profile/usage_test.go`

**Interfaces:**
- Produce `type UsageGranularity string` in `internal/core/model` with `UsageGranularityProvider`, `UsageGranularityProfile`, `UsageGranularityGroup`, and `UsageGranularityModel`; add backward-compatible `Granularity UsageGranularity` and `QuotaGroup string` fields to `model.UsageSnapshot`.
- Produce `type ProbeDescriptor struct { Sources []model.UsageSource; RequiresNetwork bool; ReadOnly bool; MinimumInterval time.Duration; Confidence model.UsageStatus; TTL time.Duration; Granularity model.UsageGranularity }`.
- Produce `type Probe interface { Descriptor() ProbeDescriptor; Probe(context.Context, model.Profile) (model.UsageSnapshot, error) }`.
- Produce `func IsAttributableModelName(snap model.UsageSnapshot) bool` and `func DisplayQuotaSubject(snap model.UsageSnapshot, group string) string`.

- [ ] **Step 1: Write the failing tests** for probe descriptors rejecting `ReadOnly == false`, for stale/unknown snapshots remaining unknown, and for `DisplayQuotaSubject` returning the group/provider when `ModelName` is empty or non-attributable.
- [ ] **Step 2: Run the focused tests to verify RED.** Run `go test ./internal/core/quota ./internal/profile -run 'Probe|ModelName|QuotaSubject' -v`; expected failure is missing contracts or the current fabricated provider model labels.
- [ ] **Step 3: Implement the minimum contracts and honesty fix.** Remove the provider-specific fallback strings in `GetQuotaDetails`; expose `ModelName` only when `snap.Granularity == model.UsageGranularityModel` and `snap.ModelName` is non-empty. Make view/event callers fall back to `QuotaGroup`, then provider ID. Preserve legacy JSON compatibility and existing `UsageSnapshot` source fields.
- [ ] **Step 4: Run focused and regression tests.** Run `go test ./internal/core/quota ./internal/profile -v`; expected result is PASS with no fabricated exact model name.
- [ ] **Step 5: Commit.** `git add internal/core/model/types.go internal/core/quota internal/profile/usage.go && git commit -m "fix(quota): preserve unknown model attribution"`

### Task 2: Add safe provider probes and privacy-safe activity/power abstractions

**Files:**
- Create: `internal/profile/capacity_probes.go`, `internal/profile/capacity_probes_test.go`
- Create: `internal/nexus/capacitymonitor/activity.go`, `activity_linux.go`, `activity_windows.go`, `activity_darwin.go`, `power.go`, `power_linux.go`, and platform tests with mutually exclusive build tags

**Interfaces:**
- Produce `type ProbeRegistry interface { For(provider string) (quota.Probe, bool); Providers() []string }`; its production implementation lives in `internal/profile`, which may import both `quota` and provider adapters without creating a cycle.
- Produce `type ActivityDetector interface { Snapshot(context.Context) (map[string]bool, error) }` where keys are normalized provider IDs only.
- Produce `type PowerState interface { LowPower(context.Context) (bool, error) }`. Linux reads only `/sys/class/power_supply`; unsupported platforms return a truthful `false, ErrPowerStateUnsupported`. Resume is detected inside the scheduler from an injected wall-clock gap, so no platform service or dependency is required and ordinary timers never request wake-up.
- Produce `func NormalizeProviderActivity(executable, resolvedPath string) (string, bool)`; it must not accept or inspect arguments/environment/session content.

- [ ] **Step 1: Write failing tests** covering official/read-only source preference, unsafe probe exclusion, executable-path normalization, no profile attribution, and a fake power state.
- [ ] **Step 2: Run RED.** Run `go test ./internal/core/quota ./internal/nexus/capacitymonitor -run 'Probe|Activity|Power' -v`; expected failure identifies missing registry and detector APIs.
- [ ] **Step 3: Implement minimal adapters** in `internal/profile` by wrapping existing cached/read-only provider paths without adding credential refresh. Implement Linux process discovery from executable names/paths only; Windows and Darwin return explicit unsupported/degraded capability until native privacy tests exist. Keep build tags mutually exclusive.
- [ ] **Step 4: Run focused tests and `go vet`.** Run `go test ./internal/core/quota ./internal/nexus/capacitymonitor -v` and `go vet ./internal/core/quota ./internal/nexus/capacitymonitor`; expected PASS.
- [ ] **Step 5: Commit.** `git add internal/core/quota internal/nexus/capacitymonitor && git commit -m "feat(quota): add safe capacity probe adapters"`

### Task 3: Persist observations, state, leases, preferences, and delivery receipts

**Files:**
- Create: `internal/nexus/store/migrations/0013_capacity_monitor.sql`, `internal/nexus/store/capacity.go`, `internal/nexus/store/capacity_test.go`
- Modify: `internal/nexus/store/store.go` when the migration runner needs the new embedded migration registered

**Interfaces:**
- Produce store records `ObservationRecord`, `CapacityStateRecord`, `MonitorLease`, `CapacityEventRecord`, `DeliveryReceipt`, `CapacityRecommendationRecord`, and `CapacityPreferences` with JSON-safe fields and no raw provider payloads.
- Produce `func (s *Store) UpsertCapacityObservation(record ObservationRecord) error`, `ListCapacityObservations() ([]ObservationRecord, error)`, `GetCapacityState(key string) (CapacityStateRecord, error)`, `UpsertCapacityState(record CapacityStateRecord) error`.
- Produce `func (s *Store) AcquireCapacityLease(owner string, now time.Time, ttl time.Duration) (MonitorLease, bool, error)`, `RenewCapacityLease(...) error`, and `ReleaseCapacityLease(...) error`.
- Produce `PutCapacityEvent`, `FindCapacityEvent(fingerprint string)`, `PutDeliveryReceipt`, `GetCapacityPreferences`, `PutCapacityPreferences`, and `PruneCapacityHistory(before time.Time)` with idempotent behavior. Pruning retains the latest observation per key and removes transition/delivery history older than 30 days.

- [ ] **Step 1: Write failing SQLite tests** for migration idempotence, observation expiry, lease takeover after expiry, duplicate fingerprint persistence, 30-day pruning that preserves the latest observation, and retention of only redacted metadata.
- [ ] **Step 2: Run RED.** Run `go test ./internal/nexus/store -run 'Capacity|Migration' -v`; expected failure is absent schema/API.
- [ ] **Step 3: Add additive migration and repository methods.** Use primary keys covering provider/profile/group/window, UTC timestamps, indexes for expiry/fingerprint/lease, and `INSERT ... ON CONFLICT` semantics. Do not alter unrelated tables or migrations.
- [ ] **Step 4: Run focused store tests twice** to prove repeat-open idempotence: `go test ./internal/nexus/store -run 'Capacity|Migration' -count=2 -v`.
- [ ] **Step 5: Commit.** `git add internal/nexus/store && git commit -m "feat(quota): persist capacity monitor state"`

### Task 4: Implement threshold state machine, adaptive scheduling, and single-leader monitor

**Files:**
- Create: `internal/nexus/capacitymonitor/types.go`, `state.go`, `scheduler.go`, `monitor.go`, `monitor_test.go`

**Interfaces:**
- Produce `type State string` constants `Unknown`, `Healthy`, `Warning`, `Critical`, `Exhausted`, `Recovered`.
- Produce `type Thresholds struct { Warning, Critical, Exhausted, Recovery float64 }` and `func (t Thresholds) Validate() error`.
- Produce `type Clock interface { Now() time.Time; NewTimer(time.Duration) Timer }`, `type JitterSource interface { Duration(time.Duration) time.Duration }`, and injected `Timer` interfaces for deterministic tests.
- Produce `func EvaluateTransition(previous State, remaining *float64, rateLimited bool, observedAt, expiresAt time.Time, now time.Time, thresholds Thresholds) (State, string)`.
- Produce `type Monitor struct{ ... }`, `func New(deps Dependencies) *Monitor`, `func (m *Monitor) Run(ctx context.Context) error`, `func (m *Monitor) Tick(ctx context.Context) error`, and `func (m *Monitor) Status(ctx context.Context) (Status, error)`.
- `Dependencies` must include `Profiles func() ([]model.Profile,error)`, `ProbeRegistry`, `ActivityDetector`, `PowerState`, `Store`, `EventPublisher`, `RecommendationProvider`, `Clock`, `JitterSource`, and `MaxConcurrency int`.

- [ ] **Step 1: Write failing tests** for every threshold transition/recovery hysteresis, independent group/window severity, unknown/stale handling, 10m/60s/30–60s/5m/backoff/resume scheduling, max two global collections, one per profile, and lease failover without duplicate transitions.
- [ ] **Step 2: Run RED.** Run `go test ./internal/nexus/capacitymonitor -run 'Transition|Schedule|Concurrency|Lease|Resume' -v`; expected failures must be behavior/API failures, not compile typos.
- [ ] **Step 3: Implement minimal state and scheduler.** Normalize nullable windows at the boundary and use exact boundaries: `remaining <= Exhausted` is exhausted, `< Critical` is critical, `<= Warning` is warning, and higher is healthy. A low state observed above `Recovery` emits `RECOVERED` once and becomes `HEALTHY` on the next trustworthy observation. Compute aggregate severity from the most severe trustworthy window, use injected time/jitter, acquire/renew one store lease, infer resume from a wall-clock gap, and publish no user event for `UNKNOWN` except deduplicated degradation.
- [ ] **Step 4: Run focused, race, and package tests.** Run `go test -race ./internal/nexus/capacitymonitor -v` and `go test ./internal/nexus/capacitymonitor`; expected PASS with bounded goroutines and no busy loop.
- [ ] **Step 5: Commit.** `git add internal/nexus/capacitymonitor && git commit -m "feat(quota): add adaptive capacity monitor"`

### Task 5: Integrate explainable resource recommendations and explicit switching

**Files:**
- Modify: `internal/nexus/resource_recommendation.go`, `internal/nexus/resource_discovery.go`
- Create/modify: `internal/nexus/resource_recommendation_test.go`, `internal/nexus/capacitymonitor/recommendation.go`, tests

**Interfaces:**
- Produce monitor-owned, dependency-neutral `ResourceKey`, `RecommendationRequest`, `Recommendation`, `RecommendationCandidate`, and `RecommendationProvider` types in `internal/nexus/capacitymonitor`; the interface is `Recommend(context.Context, RecommendationRequest) (Recommendation, error)`. The parent `nexus` package implements an adapter around `RecommendResources`, preventing `capacitymonitor -> nexus -> capacitymonitor` imports.
- Extend recommendation output with `ResetInformation`, `RejectedAlternatives`, `ContinuityImpact`, and stable `Fingerprint` fields without changing existing callers’ JSON names.
- Produce `func (n *Nexus) ConfirmCapacitySwitch(ctx context.Context, agentID, runtimeGeneration, recommendationFingerprint, provider, profile string) (*ResourceAllocation, error)` which validates generation and fingerprint before calling existing `SafeApply`/allocation flow.

- [ ] **Step 1: Write failing tests** for same-provider/group preference, capability hard rejection, known healthy outranking unknown, rejected alternatives, no eligible replacement, stale fingerprint, and explicit confirmation requirement.
- [ ] **Step 2: Run RED.** Run `go test ./internal/nexus -run 'Recommendation|CapacitySwitch' -v`; expected failure is absent fields/adapter/confirmation method.
- [ ] **Step 3: Implement minimal integration.** Feed current Agent requirements and policy into existing scoring; add continuity/reset explanation and never mutate Agent state from monitor transitions.
- [ ] **Step 4: Run Nexus regression tests.** Run `go test ./internal/nexus -run 'Resource|Scheduler|Recommendation|CapacitySwitch' -v` and `go test ./internal/nexus`; expected PASS.
- [ ] **Step 5: Commit.** `git add internal/nexus/resource_recommendation.go internal/nexus/resource_discovery.go internal/nexus/capacitymonitor && git commit -m "feat(quota): explain capacity resource recommendations"`

### Task 6: Publish canonical events and deliver browser/native notifications

**Files:**
- Modify: `internal/control/events/events.go`, `internal/control/notify/notify.go`
- Create: `internal/nexus/capacitymonitor/delivery.go`, tests; modify relevant bus/notify tests

**Interfaces:**
- Add Go constants `EventQuotaObserved`, `EventQuotaThresholdCrossed`, `EventQuotaRecovered`, `EventRecommendationChanged`, `EventMonitorDegraded`, and `EventMonitorLeadershipChanged` with stable lowercase wire values `quota.observed`, `quota.threshold_crossed`, `quota.recovered`, `recommendation.changed`, `monitor.degraded`, and `monitor.leadership_changed`.
- Produce `type CanonicalAlert struct { ID, Fingerprint, Provider, Profile, Group, Window, Severity, RemainingPercent, ResetAt, Freshness, Confidence, Subject string; Recommendation *RecommendationSummary; LockScreenSafeTitle, LockScreenSafeBody string; Actions []Action }`.
- Produce `type Channel interface { Deliver(context.Context, CanonicalAlert) error }` and `type DeliveryRouter struct{ ... }` that records deduplication and per-channel failure while allowing other channels to proceed.
- Produce `type MetricsSnapshot struct { ProbeSuccess, ProbeFailure, StaleResources, DeliveryFailure, DeduplicatedAlerts, RecommendationAvailable uint64; ProbeDurationP50, ProbeDurationP95 time.Duration; AverageCollectionInterval time.Duration }` from bounded in-memory counters/histograms with no high-cardinality account labels.
- Adapt `notify.Payload` to preserve neutral lock-screen-safe copy by default; browser delivery remains event/API-driven and does not recalculate state.

- [ ] **Step 1: Write failing tests** for event serialization, fingerprint stability, worsening/recovery/reminder emission, timezone-aware quiet-hours/snooze/channel preference filtering, safe lock-screen copy, one failed channel not blocking another, and bounded privacy-safe metrics including p50/p95 duration.
- [ ] **Step 2: Run RED.** Run `go test ./internal/control/events ./internal/control/notify ./internal/nexus/capacitymonitor -run 'Quota|Alert|Delivery|Fingerprint' -v`.
- [ ] **Step 3: Implement canonical routing** from monitor events to persisted alert records and existing native notifier, with browser clients consuming the same event shape.
- [ ] **Step 4: Run focused and regression tests.** Run the command above without `-run`, then `go test ./internal/control/...`; expected PASS.
- [ ] **Step 5: Commit.** `git add internal/control/events internal/control/notify internal/nexus/capacitymonitor && git commit -m "feat(notify): deliver canonical capacity alerts"`

### Task 7: Expose authenticated monitor API and wire explicit Agent action

**Files:**
- Modify: `internal/control/web/server.go`, `internal/control/web/handlers_nexus.go`
- Create: `internal/control/web/capacity_monitor_test.go`
- Modify: `web/src/nexus/api.ts`, `web/src/types.ts`

**Interfaces:**
- Add authenticated routes: `GET /api/v1/capacity-monitor`, `GET /api/v1/capacity-monitor/resources`, `GET /api/v1/capacity-monitor/history`, `GET|PUT /api/v1/capacity-monitor/preferences`, `POST /api/v1/capacity-monitor/alerts/{id}/ack`, `POST /api/v1/capacity-monitor/alerts/{id}/snooze`, and `POST /api/v1/agents/{id}/capacity-switch`.
- API methods return explicit JSON structs: `MonitorStatus`, `CapacityResourceState[]`, `CapacityHistoryPage`, `CapacityPreferences`, and `CapacitySwitchResult`; nullable arrays are always encoded as `[]`.
- `CapacitySwitchRequest` must include `agent_id`, `runtime_generation`, `recommendation_fingerprint`, `provider`, `profile`, and `confirmed: true`.

- [ ] **Step 1: Write failing handler tests** for authentication/CSRF, method validation, empty arrays, preference validation, acknowledgement/snooze, and stale/unconfirmed switch rejection.
- [ ] **Step 2: Run RED.** Run `go test ./internal/control/web -run 'CapacityMonitor|CapacitySwitch' -v`.
- [ ] **Step 3: Implement handlers and routes** by injecting the monitor into `NexusHandler`; delegate all state/recommendation logic to backend services and existing `SafeApply` flow.
- [ ] **Step 4: Run API tests and TypeScript checks.** Run `go test ./internal/control/web -v` and `npm --prefix web run typecheck`; expected PASS after adding typed client methods and models.
- [ ] **Step 5: Commit.** `git add internal/control/web web/src/nexus/api.ts web/src/types.ts && git commit -m "feat(web): expose capacity monitor API"`

### Task 8: Add notification-center presentation and Settings controls

**Files:**
- Modify: `web/src/notifications/notificationModel.ts`, `notificationPrefs.ts`, `InAppNotificationCenter.tsx`
- Create: `web/src/notifications/CapacityAlertCard.tsx`, `CapacityAlertCard.module.scss`, colocated tests
- Modify: `web/src/features/settings/SettingsSurface.tsx`; create `web/src/features/settings/CapacityMonitorSettings.tsx`, its module, and test when the existing surface would exceed the documented component-size heuristic
- Modify: `web/src/i18n/resources.ts` and relevant locale resources

**Interfaces:**
- Produce typed `CapacityAlert` and `CapacityMonitorPreferences` matching API JSON, plus `capacityAlertFromEvent(event): CapacityAlert | null`.
- Produce `CapacityAlertCard({ alert, onAcknowledge, onSnooze, onSwitch }: Props)` using existing `Card`, `Button`, `Switch`, `Badge`, and resource-picker primitives; no duplicate scheduler scoring.
- Extend the existing notification center history to accept canonical capacity alerts while retaining runtime attention behavior.

- [ ] **Step 1: Write failing Vitest/RTL tests** for healthy/warning/critical/exhausted/recovered rendering, group fallback when model is unknown, freshness/confidence/reset formatting via `Intl`, safe copy, keyboard actions, nullable API arrays, quiet-hour/snooze controls, and explicit switch confirmation.
- [ ] **Step 2: Run RED.** Run `npm --prefix web exec vitest run src/notifications --reporter=verbose`; expected failures identify missing typed mapper/card/Settings controls.
- [ ] **Step 3: Implement UI** with localized semantic labels, existing notification center composition, SCSS Module tokens, visible focus, reduced motion, and backend preference read/update; keep localStorage only for legacy attention preferences if needed.
- [ ] **Step 4: Run focused and frontend gates.** Run `npm --prefix web exec vitest run src/notifications src/features/settings`, `npm --prefix web run lint`, `npm --prefix web run lint:styles`, and `npm --prefix web run typecheck`; expected PASS.
- [ ] **Step 5: Commit.** `git add web/src/notifications web/src/features/settings/SettingsSurface.tsx web/src/nexus/api.ts web/src/types.ts web/src/i18n && git commit -m "feat(ui): add capacity monitor notifications"`

### Task 9: Implement opt-in, testable Linux background startup

**Files:**
- Create: `internal/nexus/capacitymonitor/autostart_linux.go`, `autostart_linux_test.go`
- Create: platform-neutral `autostart.go` plus non-Linux implementation/tests if the existing build requires them
- Modify: `internal/app/app.go`, the Nexus composition root, and `uninstall.sh` to add the headless command `nexus capacity-monitor run` and remove only the user-owned unit on uninstall; add no unrelated startup changes

**Interfaces:**
- Produce `type AutostartManager interface { Enabled(context.Context) (bool,error); SetEnabled(context.Context, bool) error; Remove(context.Context) error }`.
- Linux implementation writes/removes a user-scoped systemd unit under `$XDG_CONFIG_HOME/systemd/user/nexus-capacity-monitor.service` using an injected filesystem, command runner, and environment. Its `ExecStart` invokes the resolved Nexus executable as `capacity-monitor run`; it is idempotent, user-owned, and never enables itself during install.
- Produce `func (m *Monitor) StartBackground(ctx context.Context) error` that checks persisted consent before starting collection.

- [ ] **Step 1: Write failing Linux tests** for disabled-by-default, explicit enable/remove, idempotent unit content containing `capacity-monitor run`, restart persistence, no root/system-wide writes, headless command cancellation, uninstall cleanup, and monitor start refusal without consent.
- [ ] **Step 2: Run RED.** Run `go test ./internal/nexus/capacitymonitor -run 'Autostart|Background' -v`; expected failure is absent manager/lifecycle APIs.
- [ ] **Step 3: Implement the minimal XDG/systemd user adapter** with deterministic paths, escaped executable arguments, injected dependencies, and lifecycle cancellation on shutdown.
- [ ] **Step 4: Run Linux evidence tests.** Run `GOOS=linux go test ./internal/nexus/capacitymonitor -run 'Autostart|Background' -v` and the package race tests; expected PASS.
- [ ] **Step 5: Commit.** `git add internal/nexus/capacitymonitor internal/app uninstall.sh && git commit -m "feat(linux): add consent-gated capacity autostart"`

### Task 10: Validate integration, load bounds, release evidence, and scope

**Files:**
- Create: `internal/nexus/capacitymonitor/integration_test.go`, `load_test.go`, `release_evidence_test.go`
- Create: `DEV/validation/CAPACITY_MONITOR_LINUX_BETA.md` only as an implementation deliverable; do not modify unrelated reports

**Interfaces:**
- Integration harness must instantiate fake probes, fake detector/power state, two monitors, one temporary SQLite database, event bus, and notification recorders.
- Load test must register at least 50 profiles and simulate 30 minutes, asserting max concurrency 2, per-profile singleflight, bounded goroutines/queue, no busy loop, and no wake request.

- [ ] **Step 1: Write failing integration/load tests** for two-process leadership, restart dedupe, all required event types, web/API state, Linux notification/restart/autostart evidence, and bounded 50-profile simulation.
- [ ] **Step 2: Run RED.** Run `go test -race ./internal/nexus/capacitymonitor ./internal/control/web -run 'Integration|Load|Linux|Restart' -v`; expected failures expose incomplete wiring only.
- [ ] **Step 3: Wire the smallest missing composition-root dependencies** and record source/status/error categories without raw payloads, prompts, terminal output, environment, tokens, or unnecessary identifiers.
- [ ] **Step 4: Run release gates.** Run `gofmt -w` only on changed Go files, `make format-check`, `make lint-go`, `make test-go`, `make security`, `npm --prefix web run quality:full`, and `make quality`; expected PASS, with Linux adapter evidence captured for the beta report. A headless CI run proves notifier selection and recorder delivery, not appearance of a real desktop toast.
- [ ] **Step 5: Commit the evidence.** `git add internal/nexus/capacitymonitor internal/control/web DEV/validation/CAPACITY_MONITOR_LINUX_BETA.md && git commit -m "test(release): validate capacity monitor beta"`

## Concrete TDD Test Vectors

The implementer must start each task with a test equivalent to the following concrete vectors, run the named command, and only then add production code. These examples pin the required API and prevent tests that merely assert mock calls.

```go
func TestDisplayQuotaSubjectDoesNotInventModel(t *testing.T) {

	snap := model.UsageSnapshot{ProviderID: "gemini", ModelName: "", Status: model.UsageUnknown}
	if got := DisplayQuotaSubject(snap, "gemini"); got != "gemini" {
		t.Fatalf("subject = %q, want quota group", got)
	}
}

func TestEvaluateTransitionStaleObservationBecomesUnknown(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	remaining := 5.0
	got, _ := EvaluateTransition(Warning, &remaining, false, now.Add(-time.Hour), now.Add(-time.Minute), now, DefaultThresholds())
	if got != Unknown {
		t.Fatalf("state = %q, want UNKNOWN", got)
	}
}
```

```go
func TestLeaseTakeoverAfterExpiry(t *testing.T) {
	first, acquired, err := store.AcquireCapacityLease("one", now, time.Minute)
	if err != nil || !acquired || first.Owner != "one" { t.Fatal("first owner must acquire") }
	second, acquired, err := store.AcquireCapacityLease("two", now.Add(2*time.Minute), time.Minute)
	if err != nil || !acquired || second.Owner != "two" { t.Fatal("expired lease must be taken over") }
}
```

```go
func TestRecommendationRejectsMissingCapability(t *testing.T) {
	result := RecommendResources([]ProviderAccount{{ID: "unknown", Authenticated: true, Available: true}},
		TaskRequirements{RequiredCapabilities: []string{"resume"}}, PolicyBalanced)
	if result.Recommended != nil || result.Candidates[0].RejectionReason == "" { t.Fatal("unsupported capability must reject") }
}
```

```go
func TestCapacitySwitchRequiresCurrentFingerprintAndConfirmation(t *testing.T) {
	_, err := n.ConfirmCapacitySwitch(ctx, "agent-1", "generation-2", "old-fingerprint", "gemini", "work")
	if err == nil { t.Fatal("stale or unconfirmed switch must fail") }
}
```

```typescript
it('renders quota group when exact model is unknown', () => {
  render(<CapacityAlertCard alert={{ ...warningAlert, subject: 'gemini', modelName: undefined }} />);
  expect(screen.getByText('gemini')).toBeVisible();
  expect(screen.queryByText('Gemini 2.5 Flash / Pro')).not.toBeInTheDocument();
});
```

## Scope Check and Explicit Future Work

The delivery is intentionally split into monitoring core (Tasks 1–5), notifications and controls (Tasks 6–8), and Linux autostart/release evidence (Tasks 9–10). The desktop tray adapter is not implementable until a desktop framework is selected and separately approved; it remains a future increment consuming only the canonical event/status API. Remote mobile push is a non-goal and receives no task. No unrelated quota, scheduler, settings, styling, or platform cleanup belongs in this plan.

## Spec → Task Coverage Matrix

| Spec requirement | Covering task(s) | Verification evidence |
|---|---:|---|
| Every enabled profile/group/window monitored | 1, 2, 4, 10 | fake registry + 50-profile load test |
| Safe read-only sources and honest attribution | 1, 2 | probe descriptor and model-subject tests |
| External CLI activity without content/profile attribution | 2, 4 | detector normalization/privacy tests |
| Thresholds, hysteresis, stale/unknown behavior | 1, 4 | transition matrix and stale tests |
| Adaptive intervals, backoff, battery/suspend | 2, 4 | injected clock/power/jitter tests |
| Single leader, max concurrency, restart takeover | 3, 4, 10 | SQLite lease and two-monitor integration tests |
| Existing recommendation ranking and explanations | 5 | capability/order/rejection tests |
| Explicit Agent switch and stale fingerprint protection | 5, 7, 8 | backend/API/UI confirmation tests |
| Durable events, fingerprints, retention, receipts | 3, 6 | migration/dedupe/channel tests |
| In-app/browser/native channels from canonical backend | 6, 7, 8 | event/API/notification UI tests |
| Preferences, quiet hours, snooze, lock-screen-safe copy | 6, 7, 8 | delivery and Settings tests |
| Authenticated Web API and nullable arrays | 7, 8 | handler and TypeScript tests |
| Privacy-safe observability and failure behavior | 2, 3, 6, 10 | redaction, failure isolation, degraded tests |
| Opt-in Linux background startup | 9, 10 | XDG/systemd lifecycle evidence |
| Desktop tray future blocked by framework choice | Scope check | separate decision required before future increment |
| Remote mobile push excluded | Scope check | explicit non-goal, no implementation task |
| Unknown `ModelName` must not be fabricated | 1 | regression test in profile/quota packages |

## Release Prerequisites

- All focused TDD cycles and full Go/frontend quality gates pass on a clean Linux beta environment.
- Linux evidence demonstrates native notifier adapter selection (plus recorder delivery in headless CI), monitor restart, SQLite persistence/deduplication, autostart enable/disable in an isolated user config directory, and 50-profile/30-minute bounded load behavior. Actual desktop-toast appearance is manual evidence and is never inferred from headless CI.
- Background startup remains disabled unless the authenticated Settings flow records explicit consent; uninstall/remove leaves no user unit behind.
- Every provider probe documents source, read-only guarantee, minimum interval, confidence, expiry, and failure category; unsafe or unavailable providers remain `UNKNOWN`.
- Web API actions are authenticated and CSRF-protected where mutating; switch requests validate Agent ID, runtime generation, recommendation fingerprint, and explicit confirmation.
- Browser/native lock-screen-safe defaults, quiet hours, snooze, acknowledgements, channel failures, and notification deduplication are verified.
- i18n keys, semantic HTML, keyboard access, SCSS Module/style allowlist, strict TypeScript, nullable-array normalization, and contrast/reduced-motion checks pass.
- No raw provider payloads, prompts, command arguments, terminal output, environment values, account identifiers, filesystem paths, or tokens enter persisted events/logs.
- Desktop tray and remote push are documented as future/non-goal boundaries; no platform support is advertised without native execution evidence.
