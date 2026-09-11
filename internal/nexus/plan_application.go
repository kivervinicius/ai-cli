package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// PlanApplicationService owns the durable WorkPlan CRUD boundary shared by
// transport adapters. Intelligence, compilation and execution remain on the
// Nexus domain facade because they cross Composer, Flow and runtime concerns.
type PlanApplicationService struct {
	nexus *Nexus
}

func NewPlanApplicationService(n *Nexus) *PlanApplicationService {
	return &PlanApplicationService{nexus: n}
}

func (s *PlanApplicationService) store(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.nexus.OpenProject()
}

func (s *PlanApplicationService) Create(ctx context.Context, projectID, title, description string, phases []store.PlanPhase, facts map[string]string) (*store.WorkPlan, error) {
	st, err := s.store(ctx)
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

func (s *PlanApplicationService) Get(ctx context.Context, planID string) (*store.WorkPlan, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.GetWorkPlan(planID)
}

func (s *PlanApplicationService) List(ctx context.Context, projectID string) ([]store.WorkPlan, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListWorkPlans(projectID)
}

func (s *PlanApplicationService) Update(ctx context.Context, plan store.WorkPlan, changeSummary string) (*store.WorkPlan, *store.PlanRevision, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, nil, err
	}
	return st.UpdateWorkPlan(plan, changeSummary)
}

func (s *PlanApplicationService) Delete(ctx context.Context, planID string) error {
	st, err := s.store(ctx)
	if err != nil {
		return err
	}
	return st.DeleteWorkPlan(planID)
}

func (s *PlanApplicationService) Revisions(ctx context.Context, planID string) ([]store.PlanRevision, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListPlanRevisions(planID)
}
