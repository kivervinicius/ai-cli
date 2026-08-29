package store

import (
	"path/filepath"
	"testing"
)

func TestWorkPlan_CRUD_And_Revisions(t *testing.T) {
	tmpDir := t.TempDir()
	st, err := Open(filepath.Join(tmpDir, "test_nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	proj, err := st.CreateProject(Project{
		CanonicalPath: tmpDir,
		Name:          "Test Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 1. Create WorkPlan
	plan := WorkPlan{
		ProjectID:   proj.ID,
		Title:       "Sprint 1 Foundation",
		Description: "Initial modular foundation",
		Phases: []PlanPhase{
			{
				ID:    "phase-1",
				Title: "Phase 1: DB & API",
				Order: 1,
				Packages: []WorkPackage{
					{
						ID:                 "pkg-1",
						Title:              "Database Schema Migration",
						Goal:               "Create SQLite tables and migrate safely",
						Priority:           "CRITICAL",
						Role:               "implementer",
						AcceptanceCriteria: []string{"Tables created", "Indexes added"},
					},
				},
			},
		},
		StructuredFacts: map[string]string{
			"db_engine": "sqlite_pure_go",
		},
	}

	created, err := st.CreateWorkPlan(plan)
	if err != nil {
		t.Fatalf("create work plan: %v", err)
	}

	if created.CurrentRevision != 1 {
		t.Errorf("expected current revision 1, got %d", created.CurrentRevision)
	}

	// 2. Fetch WorkPlan
	fetched, err := st.GetWorkPlan(created.ID)
	if err != nil {
		t.Fatalf("get work plan: %v", err)
	}
	if len(fetched.Phases) != 1 || len(fetched.Phases[0].Packages) != 1 {
		t.Fatalf("unexpected phases structure: %+v", fetched.Phases)
	}

	// 3. Update WorkPlan -> creates Revision 2
	fetched.Title = "Sprint 1 Foundation (Revised)"
	updated, rev2, err := st.UpdateWorkPlan(*fetched, "Added second acceptance criteria")
	if err != nil {
		t.Fatalf("update work plan: %v", err)
	}
	if updated.CurrentRevision != 2 || rev2.Revision != 2 {
		t.Errorf("expected revision 2, got %d", updated.CurrentRevision)
	}

	// 4. List revisions
	revs, err := st.ListPlanRevisions(created.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != 2 {
		t.Errorf("expected 2 revisions, got %d", len(revs))
	}

	// 5. Create Execution Snapshot
	snap, err := st.CreateExecutionSnapshot(created.ID, rev2.ID, `{"status":"RUNNING","progress":50}`)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if snap.ID == "" {
		t.Errorf("expected snapshot ID, got empty")
	}
}
