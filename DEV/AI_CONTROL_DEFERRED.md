# AI Control — Deferred Backlog & Non-Goals

The following capabilities have been deliberately deferred from the current milestone (`v0.4.0`) to protect architecture simplicity, security boundaries, and core runtime reliability:

1. **Public Internet Hosting & Cloud SaaS**:
   - *Rationale*: Exposing developer workstations and CLI subprocesses directly to the public internet requires complex multi-tenant IAM, authentication proxies, WAFs, and DDoS protections. Access is strictly scoped to loopback, private VPNs, and SSH port forwarding tunnels.

2. **Monaco / Full IDE Code Editor**:
   - *Rationale*: AI Control is a local control plane for coding agents and terminals, answering *"what are my agents doing and how do I control them?"* It does not compete with or replace VS Code, Cursor, or JetBrains IDEs.

3. **Autonomous Task Planner / Multi-Agent Orchestrator**:
   - *Rationale*: Agent planning and reasoning belong inside the coding agents themselves (e.g. Codex, Claude Code, Gemini CLI, AGY). AI Control acts as the runtime supervisor, account rotator, and terminal multiplexer.

4. **Multi-User Enterprise RBAC**:
   - *Rationale*: AI Control is designed primarily for developer pair-programming and individual local workstation supervision. Multi-user role-based access control is deferred until multi-node Hub topologies are stabilized.

5. **Direct Public Cloud Token Proxy**:
   - *Rationale*: Storing or proxying third-party API tokens centrally introduces credential harvesting attack vectors. Credentials remain isolated in developer keyrings and official provider credential stores.
