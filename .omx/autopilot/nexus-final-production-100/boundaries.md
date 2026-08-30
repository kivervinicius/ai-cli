# Boundaries

## NEVER

- `git reset --hard`, `git checkout .`, `git clean -fd`, force push, main mutation.
- Commit/push/PR creation without explicit user authorization.
- Secrets, auth profiles, databases, browser state, or generated dependencies in git.
- Fake provider/E2E evidence or timeout increases used to hide deadlocks.

## DANGER

- Workspace persistence/migrations, SQLite schema changes, runtime ownership, protocol and provider code.
- Windows-only code must be cross-compiled and have focused tests; Linux success is not Windows proof.

## ROLLBACK

- Baseline SHA: `a1f3a573602a7c43404acf5eb1d4ac0cbc380110`.
- Preserve each user change; inspect diff before any overlapping edit.

## VERIFY

- Focused red/green tests for each fix, then Go/frontend gates and `git diff --check`.
