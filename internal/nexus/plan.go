package nexus

import (
	"context"
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
		Skills:       targetPkg.MaestroGates,
		Acceptance:   targetPkg.AcceptanceCriteria,
	}

	// Fetch dynamic Maestro skills
	client := NewMaestroClient()
	skills, _ := client.ListSkills(ctx)

	return engine.CompilePrompt(ctx, outline, plan.StructuredFacts, skills)
}

// GeneratePlanFromIntent uses Nexus Intelligence to analyze a goal and produce a structured WorkPlan.
func (n *Nexus) GeneratePlanFromIntent(ctx context.Context, projectID, goal string) (*store.WorkPlan, error) {
	engine := intelligence.NewNexusEngine(nil)
	intent, _, err := engine.Analyze(ctx, goal, projectID)
	if err != nil {
		return nil, err
	}

	outlines := intelligence.FallbackPlanOutline(intent, nil)
	var pkgs []store.WorkPackage
	for _, o := range outlines {
		pkgs = append(pkgs, store.WorkPackage{
			ID:                 "pkg_" + ids.NewRuntimeID(),
			Title:              o.Title,
			Goal:               o.Goal,
			Priority:           o.Priority,
			Status:             "READY",
			Dependencies:       o.Dependencies,
			Role:               o.Role,
			MaestroGates:       o.Skills,
			AcceptanceCriteria: o.Acceptance,
		})
	}

	phase := store.PlanPhase{
		ID:       "phase_" + ids.NewRuntimeID(),
		Title:    "Execution Phase",
		Order:    1,
		Packages: pkgs,
	}

	return n.CreateWorkPlan(ctx, projectID, intent.Intent, fmt.Sprintf("Auto-generated plan for: %s", goal), []store.PlanPhase{phase}, map[string]string{
		"scope":      intent.Scope,
		"risk_level": intent.RiskLevel,
	})
}
