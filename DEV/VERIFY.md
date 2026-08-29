# Verification: Nexus V1 (post-pending-issues)

## Status: ALL 7 PENDING ISSUES RESOLVED

### Build & Test
- `go build ./...` — clean
- `go test -race ./...` — ALL PASS
- `go vet ./...` — clean
- `npx tsc --noEmit` — clean
- `npx vitest run` — 44/44 tests PASS
- `make build` — PASS (v0.4.6)

### Pending Issues Resolution
| # | Issue | Status | Key Changes |
|---|-------|--------|-------------|
| 1 | Resources facade | ✅ | `ListResources()` real discovery, `AllocateResource()` persistence, `ResourcePicker` UI |
| 2 | Maestro synthetic state | ✅ | Honest degraded fallback, no hardcoded capabilities/recommendations |
| 3 | Update simulated | ✅ | Returns `501 Not Implemented` |
| 4 | Agent start without provider | ✅ | `ResolveStartParams()` + `REQUIRED_RESOURCE_SELECTION` flow |
| 5 | Config not reaching runtime | ✅ | `LaunchOptions` extended, full `AgentConfig` propagation |
| 6 | Terminal continuity | ✅ | `AgentTerminalBroker` integration, `runtime_changed` frames |
| 7 | Missions scaffold | ✅ | Kept as-is, documented as future work |

### Business Rules Location
All business logic lives in `internal/nexus/` (service layer). Web and TUI consume the same API endpoints. No resource selection, health checking, config propagation, or Maestro degradation logic exists in the frontend.

### Files Modified (this session)
- `internal/nexus/resource_discovery.go` (new)
- `internal/nexus/maestro.go`
- `internal/nexus/nexus.go`
- `internal/nexus/scheduler.go`
- `internal/nexus/store/agents.go`
- `internal/control/launcher/launcher.go`
- `internal/control/web/handlers_nexus.go`
- `internal/control/web/handler_terminal.go`
- `internal/control/web/broker.go`
- `internal/control/web/server.go`
- `web/src/features/agents/AgentsSurface.tsx`
- `web/src/nexus/ResourcePicker.tsx`
- `web/src/nexus/api.ts`

### Documentation Updated
- `DEV/NEXUS_V1_ARCHITECTURE.md` — resource discovery + terminal broker
- `DEV/NEXUS_V1_RESOURCE_SCHEDULER.md` — marked implemented with flow
- `DEV/NEXUS_V1_MAESTRO_INTEGRATION.md` — honest degraded fallback
- `DEV/NEXUS_V1_AGENT_MODEL.md` — updated start flow + config propagation
- `DEV/WORKLOG.md` — session entry with all changes

### Not Yet Started
- Cross-provider scoring with continuity, cooldown, and rate-limit risk
- Project-level and global policy hierarchy
- Account Handoff and Context Handoff recommendations
- Full Maestro CLI contract (`feat/nexus-contracts-v1`)
- Mission execution/orchestration (scaffold only)
- Full E2E verification with live providers
- Performance benchmarks
