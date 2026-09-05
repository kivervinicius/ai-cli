# Engineering Standards — IAPro Nexus

## Overview

This document defines the canonical engineering standards for the Nexus project. All contributors — human and AI agent — must follow these standards.

## Frontend (web/)

### Stack

- **React 19** with TypeScript 5.9 (strict mode)
- **Tailwind CSS 4** for utility styles
- **Vitest** for unit tests
- **ESLint 9** flat config with react-hooks plugin
- **Prettier** for formatting
- **Stylelint** for CSS quality

### Commands

```bash
make format          # Format frontend + Go
make format:check    # Check formatting
make lint            # Run all linters
make lint:frontend   # ESLint only
make lint:styles     # Stylelint only
make typecheck       # TypeScript check
make test:frontend   # Vitest
make quality         # All local quality gates
make quality:full    # Quality + build
```

### TypeScript

- `strict: true` is the baseline
- Prefer explicit interfaces over `any`
- `@ts-ignore` requires justification
- No `as unknown as X` without documented reason

### ESLint

- Flat config format (ESLint 9)
- react-hooks plugin enabled
- `no-explicit-any` is off globally (per-project decision)
- Unused vars with `_` prefix are allowed

### Formatting

- Prettier with single quotes, trailing commas, 100 char width
- CSS uses double quotes (override)
- Run `make format` before committing

### Testing

- Co-locate tests with source: `Component.tsx` → `Component.test.tsx`
- Use Vitest for unit tests
- 252 tests currently passing

### File Organization

```text
web/src/
├── app/           # Composition root
├── workbench/     # IDE shell (orchestration)
├── features/      # Product capabilities
├── shared/        # Truly cross-cutting code
└── assets/        # Static assets
```

### Import Rules

- `shared/` never imports from `features/`
- Features never import from each other's internals
- Use public API (index.ts) for cross-feature imports

## Backend (Go)

### Stack

- **Go 1.25** with standard library
- **golangci-lint** for static analysis
- **gofmt** for formatting

### Commands

```bash
make lint:go         # golangci-lint
make test:go         # go test -v ./...
make test:e2e        # E2E tests with race detector
make race            # Full race detector test
make security        # govulncheck
```

### Package Organization

```text
cmd/nexus/           # Entry point (config, wiring, startup)
internal/
├── app/             # CLI commands
├── control/         # Control plane (drivers, host, web, protocol)
├── core/            # Core domain (config, provider, quota, security)
├── nexus/           # Business logic (flow, mission, intelligence)
├── profile/         # Profile management
├── runtime/         # Runtime isolation
└── ...
```

### Conventions

- Entry points in `cmd/` — minimal, just wiring
- Business logic in `internal/` packages by capability
- Platform-specific code: `*_unix.go` / `*_windows.go`
- No Java-style `controllers/services/repositories` structure
- Each package owns its types and API

### Suppressions

- `//nolint` must include linter name and justification
- No generic `//nolint` without explanation

## Formatting

### Canonical Commands

| Command | What it does |
|---------|--------------|
| `make format` | Prettier + gofmt |
| `make format:check` | Check without modifying |
| `make lint` | All linters |
| `make lint:fix` | Auto-fix what's possible |
| `make quality` | Format check + lint + typecheck + tests |
| `make quality:full` | Quality + build + security |
| `make build` | Full build (frontend + Go) |

## Git Hooks

- **pre-commit**: lint-staged (ESLint, Prettier, Stylelint, gofmt)
- **commit-msg**: commitlint (Conventional Commits)

### Commit Format

```
type(scope): description

Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
Scopes: terminal, chat, flow, agents, providers, quota, workspace, settings, web, go, infra, security, ui, deps, release
```

## CI Pipeline

### Frontend Job

1. Format check
2. Typecheck
3. ESLint
4. Stylelint
5. Unit tests
6. Build
7. Bundle sync verification

### Go Jobs (Linux, Windows, macOS)

1. Gofmt check
2. Go vet
3. golangci-lint (Linux only)
4. Tests with race detector (Linux/macOS)
5. E2E tests
6. Binary build

### Snapshot Job

- GoReleaser cross-compilation
- Artifact verification

## Quality System Test

To verify the quality system is working, intentionally introduce an error in each tool and confirm the gate fails:

1. Add `any` type in TypeScript → typecheck fails
2. Add unused import → ESLint fails
3. Add invalid CSS → Stylelint fails
4. Break Go formatting → gofmt check fails
5. Add Go lint issue → golangci-lint fails

Remove the intentional error after verification.

## Coding Agent Rules

Before modifying code:

1. Understand feature ownership
2. Don't create global utils without need
3. Don't create new primitives if one exists
4. Don't ignore lint warnings
5. Don't add suppressions to pass CI

Before finishing:

1. Run `make quality`
2. Verify tests pass
3. Check formatting

Coding agents must not redefine project standards.

## Multiplatform

- Windows, Linux, and macOS are first-class targets
- Platform-specific code must be isolated in `*_unix.go` / `*_windows.go`
- CI runs on all three platforms
- Never assume Linux-only environment
