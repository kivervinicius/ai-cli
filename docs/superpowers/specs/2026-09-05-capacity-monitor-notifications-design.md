# Nexus Capacity Monitor and Notifications — Design

## Purpose

Build a lightweight, continuous capacity monitor for every CLI provider profile
registered in Nexus. The monitor detects quota pressure even when a CLI is used
outside Nexus, recommends the closest eligible replacement, and delivers useful
alerts without silently switching a running Agent or consuming excessive machine
resources.

## Goals

- Monitor every enabled `provider + profile + quota group/window` registered in
  Nexus, not only profiles attached to active Nexus Agents.
- Detect external CLI activity without reading prompts, commands, terminal output,
  environment variables, or session content.
- Classify trustworthy observations as healthy, warning, critical, exhausted, or
  recovered.
- Recommend the nearest eligible registered resource with an explainable ranking.
- Deliver in-app, browser, and native operating-system notifications from one
  canonical backend event.
- Support a future desktop tray adapter without coupling the monitor to a desktop
  framework.
- Remain opt-in for background startup and respectful of battery, network, CPU,
  memory, provider rate limits, and suspended machines.

## Non-goals

- Nexus does not inspect or control external CLI sessions.
- Nexus does not infer exact account consumption solely from a running process.
- Nexus does not switch an Agent automatically because a threshold was crossed.
- The first delivery does not include remote mobile push. Remote push requires a
  separately approved design covering subscriptions, keys, revocation, delivery
  infrastructure, and privacy.
- The first delivery does not select a desktop framework or ship a desktop app.

## Terminology and honesty rules

- **Profile:** a provider account configuration registered in Nexus.
- **Quota group:** a provider-defined model family sharing capacity, such as
  `gemini` or `claude_gpt`.
- **Window:** a capacity period such as five hours or one week.
- **Observation:** a point-in-time reading with source, timestamp, confidence, and
  expiry.
- **Exact model:** used only when the provider supplies attributable evidence.
  Otherwise UI and events name the quota group or provider, never an invented model.
- **External activity:** evidence that a provider CLI process exists. It only
  changes collection frequency and never identifies a profile by itself.
- **Unknown data:** absence of trustworthy capacity evidence. Unknown or stale data
  is never converted into zero capacity.

## Architecture

```text
Official APIs / safe CLI commands / provider-owned local files / Nexus rate limits
                                  |
                                  v
                         Read-only quota probes
                                  |
                                  v
                  Adaptive capacity monitor (single leader)
                         |                  |
                         v                  v
                  State transitions    Existing scheduler
                         |                  |
                         +--------+---------+
                                  v
                         Persistent event record
                                  |
              +-------------------+-------------------+
              v                   v                   v
          Web/in-app        native notifier     future tray adapter
```

The backend is the single authority for collection, threshold state, deduplication,
recommendation, and user preferences. Delivery channels render canonical events;
they do not independently recalculate quota state or choose a fallback.

## Component boundaries

### Read-only quota probes

`internal/core/quota` gains a probe contract implemented by provider-specific
adapters. A probe declares:

- supported observation sources;
- whether network access is required;
- whether the operation is guaranteed read-only;
- minimum supported interval;
- expected confidence and expiry semantics;
- quota granularity: provider, profile, group, or exact model.

A probe that might trigger authentication, refresh credentials with side effects,
or mutate provider configuration is not eligible for background collection. Such a
profile remains `UNKNOWN` until a safe observation becomes available.

Source preference is: official provider API, official read-only CLI command,
provider-owned local usage file, then Nexus-observed rate-limit evidence. Every
snapshot retains its actual source; sources are not blended into false precision.

### Process activity detector

A platform-specific detector reports only normalized provider IDs with a boolean
active state and observation time. It may inspect process executable names and
resolved executable paths. It must not inspect command arguments containing user
content, process environment, open files, terminal buffers, or IPC traffic.

External activity increases collection frequency for all enabled profiles of that
provider. It cannot attribute activity to an account unless a separate trustworthy
source exposes that attribution.

### Adaptive capacity monitor

`internal/nexus/capacitymonitor` owns scheduling, singleflight, backoff, state
evaluation, leader election, persistence, and event publication. Only one local
Nexus process holds the monitor lease. Other Nexus processes read the persisted
state and event stream.

The monitor uses at most two concurrent collections globally and at most one per
provider/profile. Collection intervals are:

- healthy and no observed activity: 10 minutes plus randomized jitter;
- active external CLI or active Nexus Agent: 60 seconds plus jitter;
- warning or critical: between 30 and 60 seconds, never below the probe minimum;
- battery saver or low-power mode: at least 5 minutes;
- offline or repeated failure: exponential backoff of 1, 2, 5, then 15 minutes;
- suspended machine: no wake request; evaluate once after resume with jitter.

File-backed sources should use filesystem change observation where reliable, with a
slow periodic reconciliation to recover from missed events.

### Threshold state machine

State is tracked independently for every profile, quota group, and window:

- `UNKNOWN`: no trustworthy unexpired observation;
- `HEALTHY`: more than 20% remains;
- `WARNING`: 10% through 20% remains;
- `CRITICAL`: more than 0% and less than 10% remains;
- `EXHAUSTED`: 0% remains or an attributable rate limit is confirmed;
- `RECOVERED`: a previously warning, critical, or exhausted resource is above 25%
  or has a confirmed reset-cycle observation.

The thresholds are configurable globally and overridable per provider or quota
group. Defaults are warning 20%, critical 10%, exhausted 0%, and recovery 25%.
Configuration validation enforces `0 <= exhausted < critical < warning < recovery
<= 100`.

For multiple windows, each window retains its state and the most severe trustworthy
window determines the aggregate state. Alerts name the limiting window and also
show other known windows. A transition is emitted only when severity worsens,
recovery occurs, the recommended replacement changes materially, or a configured
reminder interval elapses.

### Recommendation engine

The monitor delegates ranking to the existing Nexus resource recommendation and
scheduler code. It supplies the at-risk resource, current Agent requirements when
available, continuity context, required capabilities, health, quota confidence,
cooldowns, and project policy.

Eligible candidates are ranked in this order before normal score tie-breaking:

1. Same provider and quota/model family on another healthy profile.
2. Same provider with a task-compatible model or group.
3. Another provider with explicitly supported required capabilities.
4. Best eligible resource with continuity loss clearly stated.

Unknown quota does not outrank known healthy capacity merely because it lacks an
observed limit. A candidate with missing required capabilities is rejected. Every
recommendation contains the selected resource, confidence, reset information,
score breakdown, rejected alternatives, and continuity impact.

An active Nexus Agent may receive `Switch now`, which invokes the existing safe
runtime reconfiguration and continuity flow after explicit confirmation. External
CLI activity receives informational guidance and an optional safe command to copy;
Nexus never injects input or terminates the external process.

## Persistence and event model

SQLite stores:

- latest observation and expiry for each profile/group/window;
- current threshold state and transition timestamp;
- monitor lease owner and expiry;
- canonical alert fingerprint and delivery history by channel;
- latest recommendation and explanation;
- user acknowledgement, dismissal, snooze, and switch confirmation;
- monitor preferences and provider/group overrides.

Migrations are additive and idempotent. Retention defaults to the latest observation
plus 30 days of transitions and delivery receipts. Raw provider payloads are not
persisted.

Canonical event types are:

- `quota.observed` for internal state and diagnostics, not user notification;
- `quota.threshold_crossed` for warning, critical, and exhausted transitions;
- `quota.recovered` for confirmed recovery;
- `recommendation.changed` when the best eligible candidate changes materially;
- `monitor.degraded` when collection becomes stale or repeatedly fails;
- `monitor.leadership_changed` for operational diagnosis.

An alert fingerprint is derived from provider, profile, quota group, window,
severity, and reset cycle. This makes deduplication survive process and browser
restarts. No fingerprint includes account email, prompt content, filesystem path,
or access token.

## Notification behavior

All user-visible text is localized. A detailed alert contains:

- provider and safe profile display name;
- exact model only when attributable, otherwise quota group;
- remaining percentage and limiting window;
- reset time using the user's locale;
- observation freshness and confidence;
- recommended replacement and continuity impact;
- available actions.

The Web notification center displays full detail. Browser and native OS
notifications suppress sensitive details on lock-screen-safe mode, which is the
default. The future tray shows neutral, warning, or critical status and a short list
of affected resources. It consumes the event API and owns no scheduler logic.

User preferences include master enablement, background startup consent, thresholds,
reminder interval, quiet hours, sound, browser notifications, native notifications,
lock-screen detail, per-provider/group overrides, snooze, and pause. Background
startup is disabled by default. Exhausted alerts bypass quiet hours only when the
user explicitly enables that behavior.

## Web and control interfaces

The authenticated Web API exposes:

- aggregate monitor health and leadership;
- current resource states and latest observations;
- transition and notification history;
- current recommendation details;
- validated preference read/update operations;
- acknowledgement and snooze operations;
- explicit, confirmation-bound Agent switch operation.

The UI reuses the existing notification center, settings primitives, resource
picker, semantic tokens, SCSS Modules, and i18n resources. It does not introduce a
parallel notification center or duplicate scheduler scoring in TypeScript.

## Privacy and security

- Background monitoring requires explicit consent and can be disabled immediately.
- Provider credentials remain server-side within existing profile isolation.
- Observability events exclude prompts, terminal output, command history, raw
  provider responses, tokens, secrets, and unnecessary personal identifiers.
- Logs and diagnostic bundles expose source/status/error categories but redact
  account identifiers according to existing Nexus redaction policy.
- Web actions require the existing authenticated local session. Switch requests are
  bound to Agent ID, current runtime generation, recommendation fingerprint, and
  explicit confirmation to prevent stale-action execution.
- A recommendation is advisory until the user confirms it; monitoring cannot widen
  Agent permissions or change execution policy.

## Failure behavior

- Missing probe capability: resource remains unknown and UI explains that the
  provider does not expose safe quota telemetry.
- Stale observation: retain the last value for history, change live state to
  unknown, and emit a deduplicated degraded-monitor event.
- Network/provider failure: back off without blocking other providers.
- Corrupt local provider file: reject the observation, retain the previous valid
  history entry, and record a redacted parse category.
- Multiple Nexus processes: only the lease owner collects; lease expiry permits
  takeover without duplicate transition emission.
- No eligible replacement: alert includes reset timing and explicit rejection
  reasons instead of proposing an unsafe resource.
- Notification channel failure: record the channel failure and keep the canonical
  event available to other channels.

## Observability

The monitor records a minimal, privacy-safe taxonomy compatible with future
OpenTelemetry spans:

- probe start/end, provider, source class, outcome, duration, and freshness;
- scheduler invocation, candidate count, outcome, and confidence;
- state transition and reason;
- channel delivery success/failure and deduplication decision;
- leader acquisition/loss and backoff state.

Operational reports include probe success rate, p50/p95 duration, stale-resource
count, notification delivery failures, deduplicated alert count, recommendation
availability, and average collection interval. No claim of reduced interruptions or
better resource utilization is made until a baseline and beta feedback exist.

## Validation strategy

Unit tests use injected clocks, jitter sources, process detectors, power/network
state, probes, scheduler, store, and notification recorders. Coverage includes:

- every threshold transition and recovery hysteresis;
- independent quota groups and multiple windows;
- unknown, stale, estimated, cached, and live observations;
- process activity increasing frequency without profile attribution;
- probe minimum intervals, exponential backoff, and resume behavior;
- global concurrency of two and per-profile singleflight;
- leader failover and duplicate-event prevention;
- persistence and deduplication across restart;
- capability rejection and explainable candidate ordering;
- stale switch fingerprints and explicit confirmation;
- quiet hours, snooze, channel preferences, and lock-screen-safe copy;
- nullable API arrays, i18n coverage, keyboard access, and notification UI states.

Integration tests run fake provider probes and two Nexus instances against a
temporary SQLite database. A load test covers at least 50 registered profiles for
30 simulated minutes and proves bounded goroutine count, bounded queue depth, no
busy loop, and maximum configured concurrency. Native notification, power-state,
autostart, and future tray behavior require evidence on each supported operating
system.

## Delivery increments

### Increment 1 — monitoring core

Add probe contracts, state machine, adaptive scheduling, leader election,
persistence, canonical events, and scheduler integration. Expose read-only status
through CLI JSON and authenticated Web API. Background startup remains unavailable
until its controls exist.

### Increment 2 — user controls and notifications

Add validated backend preferences, settings UI, notification-center presentation,
browser/native delivery, quiet hours, snooze, explicit Agent switch, diagnostics,
and beta load testing. Enable opt-in background startup only on platforms with
native autostart and lifecycle tests.

### Increment 3 — desktop tray adapter

Connect a selected desktop shell to the canonical event/status interfaces. Add tray
badge/menu/actions, OS-specific startup integration, native power-state awareness,
and install/update lifecycle tests. Desktop framework selection is a separate,
bounded decision made before this increment.

Remote mobile push remains a separate future specification and is not required for
these increments.

## Acceptance criteria

- All enabled registered profiles are monitored when a safe source exists, whether
  used inside or outside Nexus.
- External process detection changes frequency but never fabricates account or
  model attribution.
- Default threshold, recovery, deduplication, and reset-cycle behavior matches this
  specification.
- The recommended replacement is eligible, capability-compatible, explainable, and
  never applied without explicit confirmation.
- Unknown/stale data never becomes a false exhaustion alert.
- A second Nexus process does not duplicate polling or notifications.
- Background startup is opt-in and all channels honor backend preferences.
- Idle monitoring has no busy loop, does not wake suspended machines, respects
  probe minimum intervals, and stays within configured concurrency.
- Web changes meet Nexus SCSS Module, semantic token, i18n, accessibility, strict
  TypeScript, nullable-array, and component-boundary standards.
- Linux beta evidence covers monitoring, native notification, restart, load, and
  installation behavior. Windows/macOS or desktop support is advertised only after
  native execution evidence exists.
