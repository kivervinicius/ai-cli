# Nexus Product Finalization Design Spec

## 1. Executive Summary

This specification consolidates the architectural decisions, safety gates, persistent models, cross-platform release guarantees, and verification matrices for the GA release of IAPro Nexus Autonomous Multi-Agent Workspace OS.

## 2. Architecture Decisions & Invariants

### 2.1 Routing & Navigation (`Wave 1`)
- **Canonical Routing**: Integration of `react-router-dom` v7 with semantic routes (`/`, `/projects`, `/settings`, `/updates`, `/p/:projectId/:surface`, `/p/:projectId/:surface/:detailId`).
- **Authority**: The URL is the single semantic authority for active project and surface. `localStorage` serves only as a transient fallback for root `/` landing.
- **Popout Isolation**: Popout windows mount in `/p/:projectId/popout/:surface` with `saveLayout = undefined`, ensuring detached popouts never overwrite the primary multi-panel project layout.

### 2.2 Layout v4 & Monotonic Persistence (`Wave 2`)
- **Envelope v4**: Canonical SQLite storage format preserving complete tree model and presentation metadata.
- **Revision Authority**: SQLite table `project_layouts` manages a monotonic integer `revision` column.
- **Optimistic Concurrency**: `PUT /api/v1/projects/:id/layout` requires the expected `revision` and fails with `409 Conflict` (`REVISION_CONFLICT`) if concurrent writes occur.

### 2.3 Execution Admission & Worktree Isolation (`Wave 3`)
- **Preflight & Admission**: Read-only `PlanPreflight` validates DAG acyclicity, step resource readiness, and autonomy requirements. Fail-closed `ExecutionAdmission` re-verifies these gates immediately prior to runner dispatch.
- **Hard Worktree Isolation**: Autonomous writers strictly require `Isolation == "worktree"`. Execution against the canonical project checkout fails closed with an actionable safety error.

### 2.4 Durable Activity & Attention Radar (`Wave 4`)
- **Single Event Bus**: The existing event bus hooks directly into SQLite `events_metadata` via `SetRecorder` on `nexus.Default()`. No concurrent or split event bus is introduced.
- **Contextual Attention**: `GlobalAttentionRadar` filters active alerts with direct navigation actions to blocking runs, unvalidated plans, or terminal sessions.

### 2.5 Hardening, Visual Regression & Accessibility (`Wave 5`)
- **Responsive Matrix**: 6 canonical viewports (`320x568`, `390x844`, `768x1024`, `1024x768`, `1280x800`, `1440x900`) verified with `document.elementFromPoint` zero-obstruction guarantees.
- **WAI-ARIA Compliance**: Accordion headers with `aria-expanded`, theme radio options with `role="radio"`, `aria-checked`, and palette swatches.
- **Density Delta**: Card container dimensions mathematically verify comfortable mode height > compact mode height.

### 2.6 Native Matrix & Packaging (`Wave 6`)
- **Cross-Platform Compilation**: Linux (`amd64`, `arm64`), Darwin (`amd64`, `arm64`), Windows (`amd64`, `arm64`).
- **Packaging**: `.tar.gz` and `.zip` archives, nFPM `.deb` and `.rpm` packages for Linux.
- **Installer Consistency**: `install.sh` and `install.ps1` archive names align with GoReleaser artifacts.

### 2.7 Signed Updates, Receipts & Rollback (`Wave 7`)
- **Ed25519 Manifests**: Signed channel manifests (`/v1/beta/manifest.json`) verified against trusted key rings over exact manifest bytes.
- **Atomic Rollback**: Automatic backup to `.bak` prior to binary replacement; JSON audit receipts stored in `receipts/update-<timestamp>.json`.
