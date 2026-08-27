# Architecture

```text
                          ai
                          │
                profile registry/defaults
                          │
            ┌─────────────┴─────────────┐
            │                           │
        Codex adapter               AGY adapter
            │                           │
       CODEX_HOME                  profile HOME
       per profile                XDG dirs per profile
            │                    D-Bus per execution
            │                  keyring per profile
            │                           │
      official codex                official agy
            │                           │
            └─────────────┬─────────────┘
                          │
                   current directory
                   same Linux UID/GID
```

## Why no token router?

The manager's abstraction boundary is environment/process isolation. OAuth remains owned by the official CLIs. This reduces breakage when token formats change and avoids putting refresh tokens into a custom token database.

## Codex adapter

```text
ai codex:a
   │
   ├─ CODEX_HOME=.../profiles/codex/a/home
   └─ exec codex [...]
```

## AGY adapter

```text
ai agy:a
   │
   ├─ HOME=.../profiles/agy/a/home
   ├─ XDG_CONFIG_HOME=profile
   ├─ XDG_CACHE_HOME=profile
   ├─ XDG_DATA_HOME=profile
   ├─ dbus-run-session
   │    └─ gnome-keyring-daemon --components=secrets
   │          └─ official agy
   └─ current working directory unchanged
```

The process UID/GID never changes.

## Adding providers later

Provider-specific code lives under `internal/provider/`. A future refactor can formalize a common interface such as:

```go
type Provider interface {
    Prepare(name string) error
    Login(name string) error
    Run(name string, args []string) error
    Status(name string) error
}
```

Candidates: Claude Code, Gemini CLI, OpenCode, Aider.
