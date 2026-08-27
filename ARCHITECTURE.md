# Architecture: AI CLI Control Plane

```text
                               ai CLI / TUI
                                    │
                         Provider Registry & Dispatcher
                                    │
        ┌──────────────┬────────────┼────────────┬──────────────┐
        │              │            │            │              │
      Codex           AGY         Claude      OpenCode        Gemini
     Adapter        Adapter      Adapter      Adapter        Adapter
        │              │            │            │              │
    CODEX_HOME    profile HOME   CLAUDE_HOME  OPENCODE_HOME  GEMINI_HOME
   per profile   XDG per profile per profile  per profile    per profile
        │        D-Bus keyring      │            │              │
        │              │            │            │              │
  official codex  official agy official claude official opencode official gemini
        │              │            │            │              │
        └──────────────┴────────────┼────────────┴──────────────┘
                                    │
                            current directory
                           same Linux UID/GID
```

---

## 1. Core Principles

- **No Central OAuth Proxy**: Authentication and refresh tokens are owned by official CLIs. `ai-cli` manages process isolation, environment variables, and credential storage paths.
- **Honest Quota Engine**: Never presents missing data as 100%. Quotas explicitly report `LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`, or `UNSUPPORTED`.
- **Smart Account Selection**: Scores candidate accounts based on capacity, health, project bindings, and priorities.
- **Automatic Fallback**: Recovers from 429 rate limits by switching to the next best account seamlessly.

---

## 2. Component Diagram

```mermaid
flowchart TB
    CLI["ai CLI / Bubble Tea TUI"] --> Dispatcher["Command Dispatcher & Controller"]
    Dispatcher --> Scheduler["Smart Account Selector"]
    Dispatcher --> Registry["Provider Registry"]
    Dispatcher --> SessionIndex["Universal Session Index"]
    Dispatcher --> SecurityEngine["Isolation & Security Engine"]

    Scheduler --> QuotaEngine["Usage & Quota Engine (TTL / Cache)"]
    Scheduler --> Cooldown["Cooldown & Rate Limit Tracker"]
    Scheduler --> Config["Config & Project Bindings Store"]

    Registry --> CodexAdapter["Codex Provider Adapter"]
    Registry --> AGYAdapter["AGY Provider Adapter"]
    Registry --> ClaudeAdapter["Claude Code Adapter"]
    Registry --> OpenCodeAdapter["OpenCode Adapter"]
    Registry --> GeminiAdapter["Gemini CLI Adapter"]

    CodexAdapter --> IsolatedRuntime["Isolated Process Runtime (TTY / Signals)"]
    AGYAdapter --> IsolatedRuntime
    ClaudeAdapter --> IsolatedRuntime
    OpenCodeAdapter --> IsolatedRuntime
    GeminiAdapter --> IsolatedRuntime
```

---

## 3. Run & Fallback Sequence

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant CLI as ai CLI
    participant Selector as AccountSelector
    participant Runtime as IsolatedRuntime
    participant Adapter as ProviderAdapter
    participant Cooldown as CooldownTracker

    User->>CLI: ai codex
    CLI->>Selector: SelectBestProfile(codex)
    Selector-->>CLI: Selected: profile-work (Capacity: 85%)
    CLI->>Runtime: RunInteractive(profile-work, args)
    Runtime->>Adapter: Execute Official CLI
    Adapter-->>Runtime: Process Exits with 429 RateLimit
    Runtime-->>CLI: Execution Result: Failure(Kind=RateLimitFailure)
    CLI->>Cooldown: RecordRateLimit(codex, profile-work, 30m)
    CLI->>Selector: SelectBestProfile(codex, exclude=[profile-work])
    Selector-->>CLI: Selected: profile-personal (Capacity: 70%)
    CLI->>User: ⚠ profile-work rate-limited. Falling back to profile-personal...
    CLI->>Runtime: RunInteractive(profile-personal, args)
    Runtime->>Adapter: Execute Official CLI
    Adapter-->>User: Interactive Session Connected
```

---

## 4. Provider Capabilities Matrix

| Provider | Profiles | Isolation | Usage | Resume Syntax | Cross-Account Resume | Auto Selection |
|---|---|---|---|---|---|---|
| **Codex** | Unlimited | `CODEX_HOME` | `LIVE` / `CACHED` | `codex resume <id>` | Yes (shared sessions) | Yes |
| **AGY** | Unlimited | D-Bus Keyring + `HOME` | `LIVE` / `CACHED` | `agy --conversation=<id>` | Yes (shared brain) | Yes |
| **Claude Code** | Unlimited | `CLAUDE_CONFIG_DIR` + `HOME` | `LIVE` / `UNKNOWN` | `claude --resume <id>` | No | Yes |
| **OpenCode** | Unlimited | `OPENCODE_CONFIG_DIR` + `XDG` | `LIVE` / `UNKNOWN` | `opencode -s <id>` | Yes | Yes |
| **Gemini CLI** | Unlimited | `GEMINI_CLI_HOME` + `HOME` | `LIVE` / `UNKNOWN` | `gemini -r <id>` | No | Yes |
