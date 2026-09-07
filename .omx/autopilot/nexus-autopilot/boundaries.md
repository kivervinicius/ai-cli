# Boundaries

- NEVER: destructive Git operations, auto-commit/push, secret exposure, production mutation, architecture rewrite, duplicate Maestro/task/plan core.
- DANGER: SQLite migrations, runtime scheduling, agent processes, worktrees, provider fallback and frontend embedded bundle.
- ROLLBACK: preserve current branch/status and revert only explicitly authored paths if required; all pre-existing dirty paths remain user-owned.
- VERIFY: focused red/green tests, full Go tests/race/vet, frontend format/type/lint/style/test/build gates, and evidence reports.
