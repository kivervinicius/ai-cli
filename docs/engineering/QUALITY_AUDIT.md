# Quality Audit — IAPro Nexus

Generated: 2026-09-05

| Area | Current | Problem | Target | Priority |
|------|---------|---------|--------|----------|
| **TypeScript** | 5.9.3, strict:true | noUncheckedIndexedAccess off, no explicit noImplicitOverride | strict + incremental strict flags | medium |
| **React** | 19.0.0 | No React-specific ESLint hooks rules | eslint-plugin-react-hooks | high |
| **ESLint** | 9.x flat config | No react-hooks plugin, no-import-duplicates off, no-explicit-any off globally | Add react-hooks, keep any-off per-file not global | high |
| **Prettier** | Missing | No formatter for frontend/JSON/MD/YAML | Add Prettier with .prettierrc | high |
| **Styles** | Tailwind CSS 4.3 + CSS files | No Stylelint, no CSS quality gates | Add Stylelint for CSS | medium |
| **Tests (FE)** | Vitest 3.x, 51 files, 252 tests | All pass, good coverage baseline | Maintain | low |
| **Go** | 1.25.0 | gofmt clean (except .worktrees), no golangci-lint | Add golangci-lint with curated linters | high |
| **golangci-lint** | Missing | No Go static analysis beyond go vet | Add .golangci.yml with focused linters | high |
| **Go formatting** | gofmt | No gofumpt evaluation, no nolintlint | Evaluate gofumpt, enforce nolint format | medium |
| **Security** | govulncheck not in CI | No govulncheck, no dependency audit in CI | Add govulncheck + npm audit | high |
| **Git hooks** | None (only samples) | No pre-commit, no commit-msg hooks | Add Husky + lint-staged | high |
| **Lint-staged** | Missing | No staged file validation | Add lint-staged config | high |
| **Commits** | No convention enforced | No conventional commits, no commit-msg hook | Add commitlint or equivalent | high |
| **CI** | GitHub Actions | Missing: format check, Stylelint, govulncheck, dependency audit, commit validation | Expand CI pipeline | high |
| **Architecture** | Partial features/ structure | components/ is flat god directory, services/ is global, ui/ is re-export barrel | Enforce feature boundaries | high |
| **Build** | Go build + web build scripts | Working, embedded bundle verified by CI | Maintain | low |
| **Multiplatform** | Windows/Linux/macOS CI | proc_unix/proc_windows, terminal_unix/windows present | Maintain, audit platform leakage | medium |
| **Suppression audit** | 21 ESLint warnings (unused vars) | Warnings are real unused code, not suppressions | Clean up unused imports/vars | medium |
