package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/nexus/intelligence"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"

	"sort"
)

// CreateWorkPlan creates a new structured engineering work plan.
func (n *Nexus) CreateWorkPlan(ctx context.Context, projectID, title, description string, phases []store.PlanPhase, facts map[string]string) (*store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("plan title is required")
	}

	plan := store.WorkPlan{
		ProjectID:       projectID,
		Title:           title,
		Description:     description,
		Phases:          phases,
		StructuredFacts: facts,
	}
	for pi := range plan.Phases {
		if plan.Phases[pi].ID == "" {
			plan.Phases[pi].ID = "phase_" + ids.NewRuntimeID()
		}
		for wi := range plan.Phases[pi].Packages {
			if plan.Phases[pi].Packages[wi].ID == "" {
				plan.Phases[pi].Packages[wi].ID = "pkg_" + ids.NewRuntimeID()
			}
		}
	}

	return st.CreateWorkPlan(plan)
}

// GetWorkPlan fetches a plan by ID.
func (n *Nexus) GetWorkPlan(ctx context.Context, planID string) (*store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.GetWorkPlan(planID)
}

// ListWorkPlans lists all work plans for a project.
func (n *Nexus) ListWorkPlans(ctx context.Context, projectID string) ([]store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListWorkPlans(projectID)
}

// UpdateWorkPlan updates a plan and records a new PlanRevision atomically.
func (n *Nexus) UpdateWorkPlan(ctx context.Context, plan store.WorkPlan, changeSummary string) (*store.WorkPlan, *store.PlanRevision, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, nil, err
	}
	return st.UpdateWorkPlan(plan, changeSummary)
}

// DeleteWorkPlan deletes a plan by ID.
func (n *Nexus) DeleteWorkPlan(ctx context.Context, planID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	return st.DeleteWorkPlan(planID)
}

// ListPlanRevisions lists all historical revisions of a plan.
func (n *Nexus) ListPlanRevisions(ctx context.Context, planID string) ([]store.PlanRevision, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListPlanRevisions(planID)
}

// CompilePackagePrompt compiles the exact scoped prompt for a single WorkPackage.
func (n *Nexus) CompilePackagePrompt(ctx context.Context, planID, phaseID, packageID string) (*intelligence.PromptCompilationResult, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	return n.compilePackagePromptFromPlan(ctx, plan, phaseID, packageID)
}

// compilePackagePromptFromPlan compiles from an immutable in-memory plan
// snapshot. Mission execution uses this function so edits to the live WorkPlan
// cannot mutate prompts for an already-approved run.
func (n *Nexus) compilePackagePromptFromPlan(ctx context.Context, plan *store.WorkPlan, phaseID, packageID string) (*intelligence.PromptCompilationResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("work plan snapshot is required")
	}
	var targetPkg *store.WorkPackage
	for pi := range plan.Phases {
		ph := &plan.Phases[pi]
		if phaseID != "" && ph.ID != phaseID {
			continue
		}
		for i := range ph.Packages {
			if ph.Packages[i].ID == packageID {
				targetPkg = &ph.Packages[i]
				break
			}
		}
		if targetPkg != nil {
			break
		}
	}
	if targetPkg == nil {
		return nil, fmt.Errorf("work package %s not found in plan %s", packageID, plan.ID)
	}

	// Maestro remains the sole authority for skill IDs. Explicit gates are part
	// of the approved contract and must never be silently dropped.
	validatedSkills, err := n.validateMaestroGatesStrict(ctx, uniqueStrings(append(append([]string(nil), targetPkg.MaestroGates...), targetPkg.MaestroSkills...)))
	if err != nil {
		return nil, err
	}
	return compileTargetPackagePrompt(ctx, plan, targetPkg, validatedSkills)
}

func (n *Nexus) validateMaestroGates(ctx context.Context, gates []string) []string {
	if len(gates) == 0 {
		return nil
	}
	available, err := NewMaestroClient().ListSkills(ctx)
	if err != nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(available))
	for _, skill := range available {
		allowed[skill] = struct{}{}
	}
	var out []string
	for _, skill := range gates {
		if _, ok := allowed[skill]; ok {
			out = append(out, skill)
		}
	}
	return out
}

func compileTargetPackagePrompt(ctx context.Context, plan *store.WorkPlan, targetPkg *store.WorkPackage, validatedSkills []string) (*intelligence.PromptCompilationResult, error) {
	engine := intelligence.NewNexusEngine(nil)
	outline := intelligence.WorkPackageOutline{
		Title: targetPkg.Title, Goal: targetPkg.Goal, Priority: targetPkg.Priority,
		Dependencies: targetPkg.Dependencies, Role: targetPkg.Role, Acceptance: targetPkg.AcceptanceCriteria,
	}
	return engine.CompilePrompt(ctx, outline, plan.StructuredFacts, validatedSkills)
}

// compilePackagePromptFromExecutionSnapshot trusts only Maestro gates already
// frozen into the approved snapshot. It never consults mutable live planning state.
func compilePackagePromptFromExecutionSnapshot(ctx context.Context, plan *store.WorkPlan, phaseID, packageID string) (*intelligence.PromptCompilationResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("work plan snapshot is required")
	}
	for pi := range plan.Phases {
		ph := &plan.Phases[pi]
		if phaseID != "" && ph.ID != phaseID {
			continue
		}
		for i := range ph.Packages {
			pkg := &ph.Packages[i]
			if pkg.ID == packageID {
				return compileTargetPackagePrompt(ctx, plan, pkg, uniqueStrings(append(append([]string(nil), pkg.MaestroGates...), pkg.MaestroSkills...)))
			}
		}
	}
	return nil, fmt.Errorf("work package %s not found in execution snapshot", packageID)
}

// ClarificationCheckpoint is the durable user-facing representation of a blocking ambiguity set.
type ClarificationCheckpoint struct {
	ID        string                       `json:"id"`
	ProjectID string                       `json:"project_id"`
	Goal      string                       `json:"goal"`
	Status    string                       `json:"status"`
	Intent    *intelligence.IntentAnalysis `json:"intent"`
	Unknowns  []intelligence.AmbiguityItem `json:"unknowns"`
	Facts     map[string]string            `json:"facts"`
}

// ClarificationRequiredError allows the REST layer to return a structured 409 instead of a generic 500.
type ClarificationRequiredError struct{ Checkpoint *ClarificationCheckpoint }

func (e *ClarificationRequiredError) Error() string {
	return "blocking clarification required before plan generation"
}

// GeneratePlanFromIntent uses a real configured Nexus Intelligence provider.
// It persists BLOCKING ambiguities and never falls back to fabricated packages.
// CLI providers use a single-shot PlanFromGoal call; OpenAI-compatible keeps multi-call.
func (n *Nexus) GeneratePlanFromIntent(ctx context.Context, projectID, goal string) (*store.WorkPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, intelligence.PlanGenerationTimeout)
	defer cancel()

	contextData, err := n.ComposerContextData(projectID)
	if err != nil {
		return nil, err
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		return nil, err
	}
	engine := intelligence.NewNexusEngine(provider).WithContextData(contextData)

	intent, unknowns, outlines, err := engine.PlanFromGoal(ctx, goal, projectID)
	if err != nil {
		return nil, err
	}

	state := &intelligence.ClarificationState{Unknowns: unknowns, StructuredFacts: map[string]string{}}
	// Non-blocking defaults are explicit decisions. Blocking items always require a user answer.
	for i := range state.Unknowns {
		item := state.Unknowns[i]
		if item.Level != intelligence.AmbiguityBlocking && item.DefaultChoice != "" {
			engine.ResolveClarification(state, item.Key, item.DefaultChoice)
		}
	}
	if hasBlockingUnknowns(state.Unknowns) {
		checkpoint, err := n.persistClarification(projectID, goal, intent, state.Unknowns, state.StructuredFacts)
		if err != nil {
			return nil, err
		}
		return nil, &ClarificationRequiredError{Checkpoint: checkpoint}
	}

	if len(outlines) == 0 {
		outlines, err = engine.GeneratePlan(ctx, intent, state.StructuredFacts)
		if err != nil {
			return nil, err
		}
	}
	return n.createPlanFromOutlines(ctx, projectID, goal, intent, outlines, state.StructuredFacts)
}

func hasBlockingUnknowns(items []intelligence.AmbiguityItem) bool {
	for _, item := range items {
		if item.Level == intelligence.AmbiguityBlocking && !item.IsResolved {
			return true
		}
	}
	return false
}

func (n *Nexus) persistClarification(projectID, goal string, intent *intelligence.IntentAnalysis, unknowns []intelligence.AmbiguityItem, facts map[string]string) (*ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	intentJSON, _ := json.Marshal(intent)
	unknownsJSON, _ := json.Marshal(unknowns)
	factsJSON, _ := json.Marshal(facts)
	row, err := st.CreateClarification(store.Clarification{
		ProjectID: projectID, Goal: goal, Status: store.ClarificationPending,
		IntentJSON: string(intentJSON), UnknownsJSON: string(unknownsJSON), FactsJSON: string(factsJSON),
	})
	if err != nil {
		return nil, err
	}
	return clarificationCheckpoint(row)
}

func clarificationCheckpoint(row *store.Clarification) (*ClarificationCheckpoint, error) {
	if row == nil {
		return nil, fmt.Errorf("clarification not found")
	}
	var intent intelligence.IntentAnalysis
	var unknowns []intelligence.AmbiguityItem
	facts := map[string]string{}
	if err := json.Unmarshal([]byte(row.IntentJSON), &intent); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.UnknownsJSON), &unknowns); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.FactsJSON), &facts); err != nil {
		return nil, err
	}
	return &ClarificationCheckpoint{ID: row.ID, ProjectID: row.ProjectID, Goal: row.Goal, Status: row.Status, Intent: &intent, Unknowns: unknowns, Facts: facts}, nil
}

func (n *Nexus) GetClarification(ctx context.Context, id string) (*ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	row, err := st.GetClarification(id)
	if err != nil {
		return nil, err
	}
	return clarificationCheckpoint(row)
}

// ResolveClarificationAndGeneratePlan records answers as durable facts, then continues
// the original analysis without re-asking already resolved questions.
func (n *Nexus) ResolveClarificationAndGeneratePlan(ctx context.Context, id string, answers map[string]string) (*store.WorkPlan, *ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, nil, err
	}
	row, err := st.GetClarification(id)
	if err != nil {
		return nil, nil, err
	}
	checkpoint, err := clarificationCheckpoint(row)
	if err != nil {
		return nil, nil, err
	}
	contextData, err := n.ComposerContextData(checkpoint.ProjectID)
	if err != nil {
		return nil, checkpoint, err
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, checkpoint.ProjectID)
	if err != nil {
		return nil, checkpoint, err
	}
	engine := intelligence.NewNexusEngine(provider).WithContextData(contextData)
	state := &intelligence.ClarificationState{Unknowns: checkpoint.Unknowns, StructuredFacts: checkpoint.Facts}
	for key, answer := range answers {
		if strings.TrimSpace(answer) != "" {
			engine.ResolveClarification(state, key, strings.TrimSpace(answer))
		}
	}
	unknownsJSON, _ := json.Marshal(state.Unknowns)
	factsJSON, _ := json.Marshal(state.StructuredFacts)
	row.UnknownsJSON, row.FactsJSON = string(unknownsJSON), string(factsJSON)
	if hasBlockingUnknowns(state.Unknowns) {
		row.Status = store.ClarificationPending
		if err := st.UpdateClarification(*row); err != nil {
			return nil, checkpoint, err
		}
		updated, _ := clarificationCheckpoint(row)
		return nil, updated, &ClarificationRequiredError{Checkpoint: updated}
	}
	outlines, err := engine.GeneratePlan(ctx, checkpoint.Intent, state.StructuredFacts)
	if err != nil {
		return nil, checkpoint, err
	}
	row.Status = store.ClarificationResolved
	if err := st.UpdateClarification(*row); err != nil {
		return nil, checkpoint, err
	}
	plan, err := n.createPlanFromOutlines(ctx, checkpoint.ProjectID, checkpoint.Goal, checkpoint.Intent, outlines, state.StructuredFacts)
	updated, _ := clarificationCheckpoint(row)
	return plan, updated, err
}

func isSchemaPlaceholder(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "...", "…", "title", "goal", "package title", "measurable criterion", "specific measurable objective":
		return true
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return true
	}
	if strings.Trim(lower, ".…") == "" {
		return true
	}
	return false
}

func sanitizeAcceptance(values []string) []string {
	var out []string
	for _, value := range values {
		if isSchemaPlaceholder(value) {
			continue
		}
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

func packagesFromOutlines(goal string, outlines []intelligence.WorkPackageOutline) []store.WorkPackage {
	goal = strings.TrimSpace(goal)
	var pkgs []store.WorkPackage
	for i, o := range outlines {
		title := strings.TrimSpace(o.Title)
		pkgGoal := strings.TrimSpace(o.Goal)
		if isSchemaPlaceholder(title) {
			if !isSchemaPlaceholder(pkgGoal) {
				title = pkgGoal
			} else if goal != "" {
				title = goal
			} else {
				title = fmt.Sprintf("Work step %d", i+1)
			}
		}
		if isSchemaPlaceholder(pkgGoal) {
			pkgGoal = goal
			if pkgGoal == "" {
				pkgGoal = title
			}
		}
		priority := strings.TrimSpace(o.Priority)
		if priority == "" || isSchemaPlaceholder(priority) {
			priority = "HIGH"
		}
		role := strings.TrimSpace(o.Role)
		if role == "" || isSchemaPlaceholder(role) {
			role = "implementer"
		}
		pkgs = append(pkgs, store.WorkPackage{
			ID: "pkg_" + ids.NewRuntimeID(), Title: title, Goal: pkgGoal, Priority: priority,
			Status: "READY", Dependencies: append([]string(nil), o.Dependencies...), Role: role,
			// Intelligence cannot mint Maestro skill/gate IDs. Maestro assignment is a separate explicit step.
			MaestroGates: []string{}, AcceptanceCriteria: sanitizeAcceptance(o.Acceptance),
		})
	}
	if len(pkgs) == 0 {
		title := goal
		if title == "" {
			title = "Work step"
		}
		pkgs = []store.WorkPackage{{
			ID: "pkg_" + ids.NewRuntimeID(), Title: title, Goal: goal, Priority: "HIGH",
			Status: "READY", Role: "implementer", MaestroGates: []string{},
		}}
	}
	byID := make(map[string]struct{}, len(pkgs))
	byTitle := make(map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		byID[pkg.ID] = struct{}{}
		key := strings.ToLower(strings.TrimSpace(pkg.Title))
		if key != "" {
			if _, exists := byTitle[key]; !exists {
				byTitle[key] = pkg.ID
			}
		}
	}
	for i := range pkgs {
		seen := map[string]struct{}{}
		var resolved []string
		for _, dep := range pkgs[i].Dependencies {
			dep = strings.TrimSpace(dep)
			if dep == "" || isSchemaPlaceholder(dep) || dep == pkgs[i].ID {
				continue
			}
			id := dep
			if _, ok := byID[dep]; !ok {
				mapped, ok := byTitle[strings.ToLower(dep)]
				if !ok || mapped == pkgs[i].ID {
					continue
				}
				id = mapped
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			resolved = append(resolved, id)
		}
		pkgs[i].Dependencies = resolved
	}
	return pkgs
}

func (n *Nexus) createPlanFromOutlines(ctx context.Context, projectID, goal string, intent *intelligence.IntentAnalysis, outlines []intelligence.WorkPackageOutline, facts map[string]string) (*store.WorkPlan, error) {
	pkgs := packagesFromOutlines(goal, outlines)
	phase := store.PlanPhase{ID: "phase_" + ids.NewRuntimeID(), Title: "Execution Phase", Order: 1, Packages: pkgs}
	mergedFacts := map[string]string{}
	for k, v := range facts {
		mergedFacts[k] = v
	}
	mergedFacts["scope"] = intent.Scope
	mergedFacts["risk_level"] = intent.RiskLevel
	return n.CreateWorkPlan(ctx, projectID, intent.Intent, fmt.Sprintf("AI-generated plan for: %s", goal), []store.PlanPhase{phase}, mergedFacts)
}

// PlanRevisionDiff is a semantic diff suitable for UI/audit without relying on
// textual JSON ordering.
type PlanRevisionDiff struct {
	FromRevision       int      `json:"from_revision"`
	ToRevision         int      `json:"to_revision"`
	TitleChanged       bool     `json:"title_changed"`
	DescriptionChanged bool     `json:"description_changed"`
	AddedPackages      []string `json:"added_packages"`
	RemovedPackages    []string `json:"removed_packages"`
	ChangedPackages    []string `json:"changed_packages"`
}

func (n *Nexus) ComparePlanRevisions(ctx context.Context, planID string, fromRevision, toRevision int) (*PlanRevisionDiff, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	fromRow, err := st.GetPlanRevision(planID, fromRevision)
	if err != nil {
		return nil, err
	}
	toRow, err := st.GetPlanRevision(planID, toRevision)
	if err != nil {
		return nil, err
	}
	var from, to store.WorkPlan
	if err := json.Unmarshal([]byte(fromRow.SnapshotJSON), &from); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(toRow.SnapshotJSON), &to); err != nil {
		return nil, err
	}
	diff := &PlanRevisionDiff{FromRevision: fromRevision, ToRevision: toRevision, TitleChanged: from.Title != to.Title, DescriptionChanged: from.Description != to.Description}
	left, right := flattenPackages(from), flattenPackages(to)
	for id, pkg := range right {
		old, ok := left[id]
		if !ok {
			diff.AddedPackages = append(diff.AddedPackages, id)
			continue
		}
		oldJSON, _ := json.Marshal(old)
		newJSON, _ := json.Marshal(pkg)
		if string(oldJSON) != string(newJSON) {
			diff.ChangedPackages = append(diff.ChangedPackages, id)
		}
	}
	for id := range left {
		if _, ok := right[id]; !ok {
			diff.RemovedPackages = append(diff.RemovedPackages, id)
		}
	}
	sort.Strings(diff.AddedPackages)
	sort.Strings(diff.RemovedPackages)
	sort.Strings(diff.ChangedPackages)
	return diff, nil
}

func flattenPackages(plan store.WorkPlan) map[string]store.WorkPackage {
	out := map[string]store.WorkPackage{}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			out[pkg.ID] = pkg
		}
	}
	return out
}

// RestoreWorkPlanRevision creates a NEW revision from a historical immutable
// snapshot; history is never rewritten or deleted.
func (n *Nexus) RestoreWorkPlanRevision(ctx context.Context, planID string, revision int) (*store.WorkPlan, *store.PlanRevision, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, nil, err
	}
	row, err := st.GetPlanRevision(planID, revision)
	if err != nil {
		return nil, nil, err
	}
	var historical store.WorkPlan
	if err := json.Unmarshal([]byte(row.SnapshotJSON), &historical); err != nil {
		return nil, nil, err
	}
	current, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, nil, err
	}
	historical.ID = current.ID
	historical.ProjectID = current.ProjectID
	historical.MissionID = current.MissionID
	historical.CurrentRevision = current.CurrentRevision
	historical.CreatedAt = current.CreatedAt
	return st.UpdateWorkPlan(historical, fmt.Sprintf("Restore from revision %d", revision))
}
