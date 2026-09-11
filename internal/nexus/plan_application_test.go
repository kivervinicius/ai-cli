package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestPlanApplicationServiceCRUDPreservesWorkPlanContracts(t *testing.T) {
	n := openTestNexus(t)
	service := NewPlanApplicationService(n)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Plans", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	plan, err := service.Create(context.Background(), project.ID, "Ship", "description", []store.PlanPhase{{Packages: []store.WorkPackage{{Title: "Build"}}}}, map[string]string{"source": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID == "" || plan.Phases[0].ID == "" || plan.Phases[0].Packages[0].ID == "" {
		t.Fatalf("created plan was not normalized: %#v", plan)
	}

	listed, err := service.List(context.Background(), project.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %#v, err = %v", listed, err)
	}

	plan.Title = "Ship safely"
	updated, revision, err := service.Update(context.Background(), *plan, "rename")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Ship safely" || revision == nil {
		t.Fatalf("update = %#v, revision = %#v", updated, revision)
	}

	revisions, err := service.Revisions(context.Background(), plan.ID)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("revisions = %#v, err = %v", revisions, err)
	}
	if err := service.Delete(context.Background(), plan.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPlanApplicationServiceRejectsCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	service := NewPlanApplicationService(n)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx, "project-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
}
