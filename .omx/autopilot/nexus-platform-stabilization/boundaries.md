# Boundaries

## NEVER

- Do not run `git reset --hard`, `git checkout --`, broad deletion or destructive cleanup.
- Do not auto-commit, auto-push, alter production services, rotate real signing keys or publish releases.
- Do not expose credentials, environment dumps, provider prompts/transcripts or raw logs.
- Do not change unrelated pre-existing files: `internal/app/app.go`, `internal/control/web/dist/bundle.css`, `internal/core/quota/quota.go`, `internal/profile/usage.go`, `internal/tui/usage_table.go` unless a later task proves ownership and records the conflict.

## DANGER

- Runtime lifecycle, filesystem identity, credential isolation, migrations, update apply and release signing are high-risk boundaries.
- All database and path migrations must be additive, transactional, collision-aware and rollbackable.
- Native behavior must be verified on the matching OS runner; cross-compilation is not support evidence.

## ROLLBACK

- Preserve current branch and starting SHA in the checkpoint.
- Keep each task in a separate Conventional Commit only when the maestro explicitly validates committing; otherwise leave changes staged/unstaged and document them.
- For migrations and updater work, use temporary stores, snapshot artifacts and failure-injection tests.

## VERIFY

- Focused tests for each changed module.
- `go test ./...`, `go vet ./...`, `make lint:go`, `make security`.
- Frontend `make web-verify` when web changes are touched.
- Native runner tests for Windows/macOS/Linux runtime work.
- Full CI evidence before any release claim.
