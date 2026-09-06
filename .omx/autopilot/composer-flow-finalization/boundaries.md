# Boundaries

- NEVER: reset/discard user changes, rewrite architecture wholesale, expose secrets, replace Go Supervisor, make Maestro a global execution gate, or auto-push.
- DANGER: migrations, runtime scheduling, provider credentials, shell/terminal execution, cross-platform behavior.
- ROLLBACK: branch `feat/autopilot-composer-flow` and worktree `.worktrees/autopilot-composer-flow`; all changes remain isolated until reviewed.
- VERIFY: focused Go tests, frontend gates, full tests, `go vet`, `git diff --check`, and final evidence reports.
