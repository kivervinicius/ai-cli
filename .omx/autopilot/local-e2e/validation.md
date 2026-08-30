# Validation

- `go test ./...` — PASS
- `go test -race ./...` — PASS
- `go vet ./...` — PASS
- `go build ./cmd/nexus` — PASS
- `bun install --frozen-lockfile` — PASS
- `bun run typecheck` — PASS
- `bun run lint` — PASS
- `bun run test` — PASS, 76/76
- `bun run build` — PASS
- `go run ./scripts/nexus-e2e-local.go --start --port 3001` — bootstrap/cleanup PASS; `NOT_AUTHENTICATED`
- `npx --yes playwright --version` — 10.9.2
