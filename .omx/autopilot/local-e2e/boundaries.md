# Boundaries

- NEVER: destructive Git commands, auto-commit/push, production deploy, real customer data, or secret output.
- DANGER: local auth/provider runtime; use only loopback and temporary Git/data.
- ROLLBACK: revert only the files shown by `git diff`; preserve the pre-existing `package.json` edit.
- VERIFY: Go/frontend gates, token scan, process/port check, and validation reports.
