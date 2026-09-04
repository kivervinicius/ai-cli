package nexus

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestComposerFinalizationCreatesPromptWithoutWorkPlan(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := n.CreateComposerSession(context.Background(), project.ID, "Help me design test coverage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.AddComposerTurn(context.Background(), session.Session.ID, "USER", "The project is a React dashboard and needs browser regression coverage."); err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), session.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Content == "" || artifact.Version != 1 {
		t.Fatalf("invalid artifact: %+v", artifact)
	}
	plans, err := st.ListWorkPlans(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("finalizing Composer must not create a Flow: %+v", plans)
	}
}
