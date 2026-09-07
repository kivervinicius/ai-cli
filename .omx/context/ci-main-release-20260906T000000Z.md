# Ralph context snapshot — CI main release

## Task statement

Make the repository CI fully green so the current product can be officially published to `main`.

## Desired outcome

All GitHub Actions jobs and release gating checks pass for the same commit, with local evidence reproducing the relevant Linux, Windows, macOS, frontend, desktop, and snapshot checks. No commit or push is performed automatically.

## Known facts/evidence

- Current branch is `feat/nexus-maximum-delivery`.
- The working tree contains many pre-existing user changes; they must be preserved.
- `DEV/HANDOFF.md` records prior failures in native Windows/macOS CI jobs.
- CI is defined in `.github/workflows/ci.yml`; release gating is in `.github/workflows/release.yml`.
- GitHub API access is currently unavailable from this environment, so remote run logs may require local reproduction or later user-side rerun.

## Constraints

- Follow repository `AGENTS.md`, engineering standards, and Orquestrador Maestro persistence rules.
- Do not commit or push automatically.
- Avoid broad unrelated refactors and destructive commands.
- Treat successful build and lint gates as completion requirements.

## Unknowns/open questions

- Exact failing remote job and test names are not available until GitHub API access works or logs are supplied.
- Native Windows/macOS execution cannot be reproduced on this Linux host; cross-compilation and platform-tagged tests are available.

## Likely codebase touchpoints

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- Platform-specific Go files under `internal/`
- `cmd/nexus-desktop`, Wails configuration, and `.goreleaser.yaml`
- Frontend lockfile/build verification under `web/`
