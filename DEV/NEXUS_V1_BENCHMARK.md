# Nexus V1 — Benchmark Study

Short, objective comparison of adjacent products (charter §97). Focus: patterns
useful for Nexus. **No implementation copied.** Licensing must be verified before
reusing any code from any of these.

| Project | Durable state | Agent lifecycle | Worktrees | Web UX | Multi-project | Remote | Quota | Security posture | Orchestration | Useful for Nexus |
|---|---|---|---|---|---|---|---|---|---|---|
| CompozyOS | app/DB | app-centric | — | Web | yes | — | — | account-based | — | Info architecture; not an agent runtime |
| OpenFang | store | agent OS | yes | TUI/Web | yes | — | — | — | yes | **anti-pattern guard**: Nexus must NOT become a general agent OS (§163) |
| Dante CLI | local files | sessions | yes | TUI | yes | — | — | — | — | terminal multiplexing ergonomics |
| Agent Deck | local store | per-agent sessions | — | Web | yes | — | — | — | — | persistent-agent concept (validate before reuse) |
| Orca | local | agent processes | — | TUI | — | — | — | — | — | process ownership patterns |
| ccswap | local | switch profiles | — | CLI | — | — | — | — | — | account/profile switching ergonomics |
| CLI-Manager | config files | — | — | CLI | — | — | — | — | — | provider CLI wrapper patterns |
| CCManager | local | sessions | — | — | yes | — | — | — | — | session index patterns |
| Claude Squad | local | squad of agents | — | — | yes | — | — | — | yes | multi-agent assignment (informs Mission Beta) |

## Key takeaways for Nexus V1

1. **Durable SQLite product state + live registry split** is the mainstream, sound
   pattern (matches Compozy/Agent Deck) — Nexus follows it.
2. **Persistent Agent over process identity** is the differentiator (Agent Deck) —
   Nexus implements it natively with RuntimeGenerations.
3. **Do not become a general agent OS** (OpenFang) or IDE (VS Code class) — Nexus is
   a specialized control workspace for coding CLIs (§162-163).
4. **No token proxy / no LLM gateway** — providers own auth/inference (§164).
5. Web-first with xterm.js terminals is the expected modern UX; TUI phased out.

**License note:** every project above was reviewed only for conceptual patterns;
no code was copied. Any future reuse requires license verification.
