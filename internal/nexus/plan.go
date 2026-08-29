package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/nexus/intelligence"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
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

	var targetPkg *store.WorkPackage
	for _, ph := range plan.Phases {
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
		return nil, fmt.Errorf("work package %s not found in plan %s", packageID, planID)
	}

	engine := intelligence.NewNexusEngine(nil)
	outline := intelligence.WorkPackageOutline{
		Title:        targetPkg.Title,
		Goal:         targetPkg.Goal,
		Priority:     targetPkg.Priority,
		Dependencies: targetPkg.Dependencies,
		Role:         targetPkg.Role,
		Acceptance:   targetPkg.AcceptanceCriteria,
	}

	// Only explicitly assigned and currently advertised Maestro skill IDs are
	// compiled into execution prompts. Nexus never injects the full skill catalog.
	var validatedSkills []string
	if len(targetPkg.MaestroGates) > 0 {
		client := NewMaestroClient()
		available, _ := client.ListSkills(ctx)
		allowed := make(map[string]struct{}, len(available))
		for _, skill := range available {
			allowed[skill] = struct{}{}
		}
		for _, skill := range targetPkg.MaestroGates {
			if _, ok := allowed[skill]; ok {
				validatedSkills = append(validatedSkills, skill)
			}
		}
	}

	return engine.CompilePrompt(ctx, outline, plan.StructuredFacts, validatedSkills)
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
func (n *Nexus) GeneratePlanFromIntent(ctx context.Context, projectID, goal string) (*store.WorkPlan, error) {
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		return nil, err
	}
	engine := intelligence.NewNexusEngine(provider)
	intent, unknowns, err := engine.Analyze(ctx, goal, projectID)
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

	outlines, err := engine.GeneratePlan(ctx, intent, state.StructuredFacts)
	if err != nil {
		return nil, err
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
	provider, err := n.ConfiguredIntelligenceProvider(ctx, checkpoint.ProjectID)
	if err != nil {
		return nil, checkpoint, err
	}
	engine := intelligence.NewNexusEngine(provider)
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

func (n *Nexus) createPlanFromOutlines(ctx context.Context, projectID, goal string, intent *intelligence.IntentAnalysis, outlines []intelligence.WorkPackageOutline, facts map[string]string) (*store.WorkPlan, error) {
	var pkgs []store.WorkPackage
	for _, o := range outlines {
		pkgs = append(pkgs, store.WorkPackage{
			ID: "pkg_" + ids.NewRuntimeID(), Title: o.Title, Goal: o.Goal, Priority: o.Priority,
			Status: "READY", Dependencies: o.Dependencies, Role: o.Role,
			// Intelligence cannot mint Maestro skill/gate IDs. Maestro assignment is a separate explicit step.
			MaestroGates: []string{}, AcceptanceCriteria: o.Acceptance,
		})
	}
	phase := store.PlanPhase{ID: "phase_" + ids.NewRuntimeID(), Title: "Execution Phase", Order: 1, Packages: pkgs}
	mergedFacts := map[string]string{}
	for k, v := range facts {
		mergedFacts[k] = v
	}
	mergedFacts["scope"] = intent.Scope
	mergedFacts["risk_level"] = intent.RiskLevel
	return n.CreateWorkPlan(ctx, projectID, intent.Intent, fmt.Sprintf("AI-generated plan for: %s", goal), []store.PlanPhase{phase}, mergedFacts)
}
