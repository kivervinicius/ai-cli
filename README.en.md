<p align="center">
  <a href="https://github.com/IAPro-Community">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="nexus-logo-dark.png">
      <img src="nexus-logo.png" alt="IAPro Nexus Logo" width="380">
    </picture>
  </a>
</p>

<p align="center">
  <a href="https://github.com/IAPro-Community"><img src="https://img.shields.io/badge/Organization-IAPro--Community-blueviolet?style=for-the-badge&logo=github" alt="IAPro Community"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-green.svg?style=for-the-badge" alt="License"></a>
  <a href="https://kernel.org"><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Platform"></a>
  <img src="https://img.shields.io/badge/Providers-Codex%20%7C%20AGY%20%7C%20Claude%20%7C%20OpenCode%20%7C%20Gemini%20%7C%20Cursor-7C3AED?style=for-the-badge" alt="Supported Providers">
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português</a> &nbsp;|&nbsp; <strong>🇬🇧 English</strong> &nbsp;|&nbsp; <a href="README.es.md">🇪🇸 Español</a>
</p>

<h3 align="center">
  ⚡ IAPro Nexus — Web-first Workspace for Coding Agents · Powered by Maestro
</h3>

<p align="center">
  <i>A project from the <strong><a href="https://github.com/IAPro-Community">IAPro Community</a></strong> ecosystem for Agentic Software Engineering</i>
</p>

---

# IAPro Nexus Manual

This manual covers **beginner to advanced**, in plain language for newcomers and
with real technical depth for terminal veterans.

---

## 1. What is IAPro Nexus?

**IAPro Nexus** (canonical binary `nexus`, with `ai` alias for full backward compatibility) is a **local
control workspace for coding agents** — the AI assistants that work inside your
terminal (Codex, Claude Code, Gemini CLI, OpenCode, AGY, Cursor Agent).

In short, Nexus does three things:

1. **Organizes by Projects** — every code project becomes the root of your work,
   with its own agents, sessions, terminals and settings.
2. **Keeps persistent Agents** — an "Agent" (e.g. *Backend Developer*) survives
   restarts, account changes and even provider switches. What changes is the
   "runtime generation" (the concrete execution), not the agent's identity.
3. **Web-first** — the official interface is a local web dashboard
   (`nexus web`) with real terminals (xterm.js) inside the browser. No giant
   mandatory terminal, nothing to install beyond a single binary.

Nexus is **Powered by Maestro**: the [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
is the authority on method, skills, risk and quality gates. Nexus executes the
real work (processes, terminals, accounts, quotas) without duplicating Maestro's
knowledge.

> **Honesty model:** Nexus never renders "intent as fact". If you asked for
> *stop*, the state shows `STOPPING` until confirmed. Session continuity is only
> marked `VERIFIED` when actually verified.

---

## 2. Installation

### Requirements

- **OS:** Linux, macOS or Windows (runtime, not just compilation).
- **Go 1.25+** only if building from source.
- **Providers:** official CLIs (`codex`, `claude`, `gemini`, `opencode`, `agy`,
  `cursor`) are auto-detected on `PATH`.

### Option A — Pre-built binary (recommended)

```bash
# Linux / macOS (amd64 or arm64)
curl -fsSL https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.sh | bash
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.ps1 | iex
```

The installer downloads the latest release binary and falls back to building from
source if needed.

### Option B — Build from source

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
make build          # produces ./bin/nexus
sudo make install   # optional: install to /usr/local/bin
```

### Verify

```bash
nexus version
# IAPro Nexus 0.4.1 (linux/amd64, go1.25.0) commit <sha> built <date>
nexus doctor
```

---

## 3. Getting started (beginner guide)

### 3.1 Open the Web dashboard

```bash
nexus web
# or the classic alias:
nexus control web
```

It prints a URL like:

```
Bootstrap: http://127.0.0.1:PORT/?token=...one-time-secret...
```

Open that URL in your browser. It exchanges the token for a secure session cookie
and redirects to the dashboard. The dashboard **only listens on `127.0.0.1`**
(loopback) — no public exposure by default.

> 💡 No auto-open? Use `ai control web --no-open` and copy the URL yourself.

### 3.2 Create a Project

In the **Projects** section (or Overview):

1. Enter the path of an existing code folder (e.g. `~/my-project`).
2. Click **Add**.
3. The project appears with name, slug and Maestro mode (`ASSIST` by default).

> The project gets a **stable ID** (e.g. `prj_01J...`). Two projects with the same
> base folder (`/home/a/api` and `/home/b/api`) are **different** projects.

### 3.3 Create an Agent

Inside the project:

1. Enter a name (e.g. `Backend Developer`) in **Create Agent**.
2. The agent appears with state `STOPPED`.

### 3.4 Start the Agent and open a Terminal

1. Click **Start**. Nexus launches a supervised *runtime generation* (persistent
   process) and the state goes to `WORKING`.
2. Click **Terminal** to open a real terminal (xterm.js) attached to the agent.
3. Type normally — it is the provider's terminal, inside the browser.

### 3.5 Stop and Recover

- **Stop** ends the current runtime generation.
- If the machine reboots or the process dies, the agent **keeps existing** (it is
  persistent), but the runtime does not. The state becomes `RECOVERABLE`.
  Click **Recover** to attempt resuming the session (or start a new one, clearly
  and honestly).

### 3.6 Navigation & shortcuts

- **`Ctrl/Cmd + K`**: command palette (open project, start agent, etc.).
- Side menu: **Overview · Projects · Agents · Resources · Maestro · Sessions ·
  Settings** (some sections arrive in later Gates) and the **Legacy** area
  (Runtimes · Providers · Events).

---

## 4. Concepts (the mental model)

| Concept | What it is | Why it matters |
|---|---|---|
| **Project** | The root of the domain. Everything belongs to a project. | Sessions, agents and layouts stay organized per project. |
| **Persistent Agent** | A stable identity (`agt_01J...`). | Survives restart, account/provider/model changes and terminal reconnects. |
| **Runtime generation** | One concrete execution of the agent (a process). | The agent is "permanent"; the generation is temporary. |
| **Config revision** | Immutable version of the agent's config. | Enables safe config rollback. |
| **Continuity** | How the session was resumed (honest). | `LIVE_SAME_RUNTIME`, `NATIVE_RESUME_VERIFIED/UNVERIFIED`, `CONTEXT_RECOVERED_NEW_SESSION`, etc. |
| **Maestro** | Method/process layer (community). | Recommends skills, process, risk and verification — Nexus does not duplicate it. |
| **Honest state** | What is *observed*, not what is *wanted*. | No "STOPPED" before it's true, no "VERIFIED" without verification. |

### Continuity — reading the states

| State | Meaning |
|---|---|
| `LIVE_SAME_RUNTIME` | Same process, connected now. |
| `REATTACHED_SAME_RUNTIME` | Reconnected to the same process (browser closed and came back). |
| `NATIVE_RESUME_VERIFIED` | Resumed and the provider **confirmed** the session. |
| `NATIVE_RESUME_UNVERIFIED` | Resumed with the provider session, but confirmation is **not** possible (honesty). |
| `CONTEXT_RECOVERED_NEW_SESSION` | Context recovered in a **new session** (e.g. provider switch). |
| `CONTINUITY_FAILED` | The resume failed. |

> The AI is never presented as verification: Nexus does not copy a session ID to
> "prove" continuity. When the provider cannot verify, the state is `UNVERIFIED`
> — on purpose.

---

## 5. CLI reference

The canonical command is `nexus`, with the `ai` alias preserved for full backward compatibility.

### Main commands

```bash
nexus                  # Web Workspace OS (default)
nexus web              # Explicitly launch the Web Workspace OS
nexus start codex      # Start a supervised runtime & attach terminal
nexus stop <id>        # Gracefully stop a supervised runtime
nexus ps               # List active supervised runtimes
nexus attach <id>      # Reconnect terminal to an existing runtime
nexus handoff <id> ... # Account handoff (same provider)
nexus continue <id> ...# Context handoff (new provider, NEW session)
nexus version          # Version, commit, build, go, platform
nexus doctor           # Full diagnostics of runtimes, keyrings & Maestro
nexus providers        # Detected providers + honest capabilities
nexus usage            # Real-time quota/usage (honest status)
nexus profiles         # Configured profiles/accounts
```

### Classic mode (direct provider launch)

```bash
nexus codex:work       # Codex on the "work" profile
nexus codex:auto       # Codex with automatic account selection
nexus claude           # Claude Code
nexus gemini           # Gemini CLI
nexus opencode         # OpenCode
nexus agy              # AGY / Antigravity
```

### Universal Canonical Aliases & Flag Interoperability

Nexus normalizes the most popular community CLI flags and translates them transparently to each provider's native options, both under supervised mode (`nexus start <provider>`) and direct invocation (`nexus <provider>`). **Native CLI flags remain 100% supported without any conflict.**

| Canonical Alias | Description | AGY Translation | Codex Translation | Claude Translation |
|---|---|---|---|---|
| `--yolo` or `-y` | Auto-approve permissions & bypass prompts | `--dangerously-skip-permissions` | `--dangerously-bypass-approvals-and-sandbox` | `--dangerously-skip-permissions` |
| `--continue` or `-c` | Continue most recent conversation | `--continue` | `resume --last` | `--continue` |
| `--resume <id>` or `-r <id>` | Resume a specific session by ID | `--conversation=<id>` | `resume <id>` | `--resume <id>` |
| `--print` or `-p` | Non-interactive mode (print output) | `--print` | `exec` | `--print` |
| `--effort <level>` | Reasoning effort (low, medium, high) | `--effort <level>` | `-c model_reasoning_effort="<level>"` | — |
| `--plan` | Start session in planning mode | `--mode plan` | — | — |
| `--accept-edits` | Start session in accept edits mode | `--mode accept-edits` | — | — |

Examples:
```bash
nexus agy --yolo                   # Dispatches agy with --dangerously-skip-permissions
nexus start codex --yolo           # Starts supervised runtime with sandbox bypass
nexus codex -c                     # Continues most recent Codex conversation immediately
nexus agy --resume 0192a...        # Connects directly to the specified conversation
```

#### Integrated Merged Help (`nexus <provider> --help`)

To inspect options for any provider with visibility of all supported Nexus aliases, run:
```bash
nexus agy --help       # or: nexus help agy
nexus codex --help     # or: nexus help codex
nexus claude --help    # or: nexus help claude
```
Nexus prints a prominent comparison table displaying the **canonical aliases applicable to that tool** followed immediately by the **complete official CLI help**.

#### Custom User Aliases

You can define additional custom aliases in `~/.config/nexus/config.json`:
```json
{
  "flag_aliases": {
    "--fast": {
      "agy": ["--effort", "low"],
      "codex": ["-c", "model_reasoning_effort=\"low\""]
    }
  }
}
```

### `/nexus` slash commands inside a supervised runtime

```text
/nexus status    /nexus usage    /nexus accounts    /nexus handoff codex:work
/nexus continue  /nexus detach   /nexus stop        /nexus help
```

- `/nexus ...` (and the `/ai ...` alias) is **intercepted by Nexus** (never leaks to the provider — zero bytes).
- `/help`, `/model`, `/resume` pass **normally to the provider**.
- To send a literal `/nexus` to the provider, type `//nexus ...`.

---

## 6. Providers and capabilities (full transparency)

| Provider | Control mode | Resume | Honest note |
|---|---|---|---|
| Codex | `TERMINAL` | Yes (not live-verified) | Structured events/approvals are **not** advertised as `SUPPORTED` — they would require an `app-server` adapter that does not exist in this version. |
| Claude Code | `TERMINAL` | Yes | — |
| Gemini CLI | `TERMINAL` | Yes | — |
| OpenCode | `TERMINAL` | Yes (not live-verified) | Same honest note as Codex about structured events. |
| AGY / Antigravity | `TERMINAL` | Yes | Dedicated credential isolation. |
| Cursor Agent | `TERMINAL` | — | Multi-path detection. |
| `fake` | `TERMINAL` | Yes | **Test** provider (hidden from UI), used for E2E and to experiment without spending quota. |

Golden rule: **effective capability = provider supports ∧ Nexus implements ∧
platform supports ∧ version compatible ∧ runtime probe passes.** No fake buttons,
no inflated claims. An "Allow?" prompt in the terminal is **not** "programmatic
approval".

---

## 7. Quota & usage — no lies

Nexus shows quota with an **honest status**:

```text
LIVE · CACHED · ESTIMATED · UNKNOWN · RATE_LIMITED · UNAVAILABLE
```

`UNKNOWN` **never** becomes 100%. Each reading shows its source and freshness:

```bash
nexus usage
nexus usage codex --json
```

Inside a runtime: `/nexus usage`.

---

## 8. Accounts, profiles & credential isolation

Each profile has its own `HOME`, `XDG_DATA_HOME`, `XDG_CONFIG_HOME` and — for AGY —
a private D-Bus session and dedicated keyring. OAuth tokens, API keys and
credentials stay isolated in the profile folder with restricted permissions.

> ⚠️ **Correct terminology:** this is **credential isolation / isolated profile**,
> **not** a "hermetic sandbox". The process runs with the same user permissions;
> what is isolated is the config/credential state.

Presets: `developer` (shares dotfiles/git), `strict` (full isolation), `compat`.

---

## 9. Handoff — switch account or provider safely

### Account Handoff (same provider, another account/profile)

```bash
nexus handoff <runtime-id> codex:work
```

It is **transactional**: preflight → checkpoint (barrier) → source quiesce →
**stop barrier** (target does not start while the source could still write the
same session) → target start → **continuity verification** (the resume command
must actually reference the session) → lineage update. Failures land in
`FAILED_SAFE` / `ROLLBACK`.

### Context Handoff (different provider → NEW session)

```bash
nexus continue <runtime-id> --with claude
```

Never called "resume": it is a **new session** fed by a safe checkpoint
(workspace, git branch, status, diff summary) plus a kickoff prompt. Secrets are
removed by a central redaction pipeline.

---

## 10. Remote access (private and secure)

The dashboard listens on loopback by default. To access from another machine:

### Via SSH tunnel (recommended)

```bash
# On machine A (where agents run):
nexus web --no-open

# On machine B:
ssh -N -L 8080:127.0.0.1:<PORT> user@machine-a
# open http://127.0.0.1:8080 and use the bootstrap token
```

### Via private VPN (Tailscale/WireGuard/corporate)

```bash
nexus web --listen <private-ip> --remote
```

`--remote` is an **explicit opt-in**. Public addresses (`8.8.8.8`, etc.) are
**refused** (error, not a warning). The CGNAT range `100.64.0.0/10` (Tailscale
etc.) is treated as a private network.

---

## 11. Security

| Control | State |
|---|---|
| Loopback default (`127.0.0.1` / `::1`) | ✅ |
| Public bind | ❌ refused (not just a warning) |
| Private bind | ✅ only with explicit `--remote` |
| One-time cryptographic bootstrap token | ✅ |
| `HttpOnly` + `SameSite=Strict` session cookie | ✅ |
| CSRF on state-changing REST | ✅ |
| `Origin` validation on REST and WebSocket | ✅ |
| Authenticated WebSocket | ✅ |
| No wildcard CORS | ✅ |
| CSP + `nosniff` + `Referrer-Policy` + `frame-ancestors 'none'` | ✅ |
| Terminal/provider never rendered as raw HTML | ✅ |
| Project path canonicalization (Abs → EvalSymlinks) | ✅ |
| Per-project IDOR isolation (agent of A inaccessible via B) | ✅ |
| Secret redaction (keys, tokens, JWT, cookies, `.env`, PEM) | ✅ |
| Bounded IPC framing (no unbounded allocation) | ✅ |
| Protocol version enforcement | ✅ |

---

## 12. Data & state

Nexus uses **local SQLite** (100% Go driver, no CGO, single portable binary) for
durable product state:

```text
projects · agents · agent_revisions · runtime_generations · lineage ·
events_metadata · maestro_advice · verification_evidence · project_layouts
```

The file lives at `<data-dir>/nexus.db`. Live runtime state (PIDs, sockets) stays
in the in-memory/on-disk registry (`runtimes.json`).

**What SQLite NEVER stores:** API keys, OAuth tokens, `auth.json`, cookies,
provider secrets, private keys, full terminal transcripts.

### Paths

| Purpose | Default location |
|---|---|
| Data (SQLite, registry, logs) | Linux/macOS: `~/.local/share/ai-manager` · Windows: `%LOCALAPPDATA%` |
| Config | `~/.config/ai-manager` (respects `XDG_CONFIG_HOME`) |
| Override | env `AI_CLI_DATA_DIR`, `AI_CLI_CONFIG_DIR`, `AI_CLI_STATE_DIR` |

---

## 13. Platform targets and release evidence

The code and pipeline target Linux, Windows and macOS on amd64/arm64. The
mandatory CI matrix runs Go 1.25, tests, native runtime E2E and builds on
`ubuntu-latest`, `windows-latest` and `macos-latest`; the GoReleaser snapshot only
runs after all three jobs succeed.

| Target platform | Runtime covered by CI | Release artifact | Evidence in this local copy |
|---|---|---|---|
| Linux amd64/arm64 | PTY, socket, SessionHost and Web | `tar.gz` | Frontend + available offline Go units; full Go 1.25 suite requires CI |
| Windows amd64/arm64 | ConPTY, Named Pipe, SessionHost and Web | `.zip` | requires `windows-latest` job |
| macOS amd64/arm64 | PTY, socket, SessionHost and Web | `tar.gz` | requires `macos-latest` job |

**Release rule:** only advertise a platform as validated for a version after its
native job is green. Cross-compilation alone is not runtime evidence.

---

## 14. Troubleshooting (FAQ)

**The dashboard won't open.**
```bash
ai doctor
```
Make sure the port is free and the bootstrap token URL was used.

**`could not open a new TTY`**
The optional compatibility TUI (`nexus --tui` or `nexus control`) needs a real terminal. Use `nexus`/`nexus web` in TTY-less sessions.

**The agent is `RECOVERABLE`.**
The process died (reboot/crash). The agent is intact — click **Recover**.

**`refusing to bind to public address`**
You tried `--listen <public-ip>`. Use an SSH tunnel or `--remote` with a private IP.

**Quota shows `UNKNOWN`.**
The provider does not expose the source. That is honesty, not a bug.

**The agent terminal disconnected.**
Browser closed? The runtime keeps running. Reopen and the terminal reconnects
(bounded replay). If the runtime died, the state becomes `RECOVERABLE`.

**Need help with Maestro.**
Maestro is the sibling repo [Orquestrador-Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro).
In Nexus, Maestro is optional. If the integration is unavailable, the state is `MAESTRO_DEGRADED`; Nexus does not fabricate skills, gates or recommendations.

---

## 15. Current candidate capability set

The current candidate integrates: Web-first Workspace OS, Direct work without a
Mission, persistent Agents, supervised/reconnecting terminals, Safe Apply,
resource/quota selection, explicit Intelligence, degradable Maestro integration,
versioned WorkPlans, blocking clarifications, durable Mission Runner, DAG/parallel
execution, per-package worktrees, independent review, governed remediation,
scheduling and Take Control/Return to Mission.

This **does not replace release gates**. Before publishing a version, run the full
matrix in `.github/workflows/ci.yml` and the procedure in
`DEV/NEXUS_FINAL_LOCAL_VALIDATION_PROMPT.md`.

---

## 16. Development

```bash
# Backend
go vet ./...
go test -race ./...

# Frontend (Bun is the canonical package manager; lockfile at web/bun.lock)
cd web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run test
bun run build      # produces the embedded frontend (internal/control/web/dist)

# Release snapshot
goreleaser release --snapshot --clean
```

> The final binary is **single** — the frontend is embedded in the Go binary.
> Node/Bun are not required on the end-user machine.

---

## 17. License

Apache-2.0. A project of the **IAPro Community**.

---

*IAPro Nexus · Powered by Maestro · Community-first. If a feature says "arrives
in Gate N", it has not been built yet — Nexus does not promise what does not
exist.*
