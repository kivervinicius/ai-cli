package store

import (
	"path/filepath"
	"testing"
	"time"
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

	// 6. Read immutable revision/snapshot back for Mission Runner restart.
	gotRev, err := st.GetPlanRevision(created.ID, 2)
	if err != nil {
		t.Fatalf("get plan revision: %v", err)
	}
	if gotRev.ID != rev2.ID || gotRev.SnapshotJSON == "" {
		t.Fatalf("unexpected revision: %+v", gotRev)
	}

	gotSnap, err := st.GetExecutionSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get execution snapshot: %v", err)
	}
	if gotSnap.RevisionID != rev2.ID || gotSnap.StateJSON != snap.StateJSON {
		t.Fatalf("unexpected execution snapshot: %+v", gotSnap)
	}
}

func TestMissionSchedule_AT_BecomesReady(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "schedule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Scheduled"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Ship", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Second)
	created, err := st.CreateMissionSchedule(MissionSchedule{PlanID: plan.ID, ProjectID: proj.ID, Mode: "AT", ScheduledFor: &when})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := st.ListReadyMissionSchedules(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 || ready[0].ID != created.ID {
		t.Fatalf("schedule not ready: %+v", ready)
	}
}

func TestMissionScheduleBindsRunID(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "schedule-run.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Scheduled Run"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Ship", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Second)
	item, err := st.CreateMissionSchedule(MissionSchedule{PlanID: plan.ID, ProjectID: proj.ID, Mode: "AT", ScheduledFor: &when})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.BindMissionScheduleRun(item.ID, "run-real"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-real" || got.Status != ScheduleRunning {
		t.Fatalf("run binding not persisted: %+v", got)
	}
}

func TestMissionRunSaveIsFencedByLeaseToken(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "fencing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Fence"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Plan", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	rec := MissionRunRecord{ID: "run-fenced", PlanID: plan.ID, ProjectID: proj.ID, State: "EXECUTING", PayloadJSON: `{"id":"run-fenced"}`}
	if err := st.UpsertMissionRun(rec); err != nil {
		t.Fatal(err)
	}
	first, err := st.AcquireMissionLease(rec.ID, "worker-a", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	second, err := st.AcquireMissionLease(rec.ID, "worker-b", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	second.State = "PAUSED"
	second.PayloadJSON = `{"id":"run-fenced","state":"PAUSED"}`
	if err := st.UpsertMissionRun(*second); err != nil {
		t.Fatal(err)
	}
	first.State = "FAILED"
	first.PayloadJSON = `{"id":"run-fenced","state":"FAILED"}`
	if err := st.UpsertMissionRun(*first); err == nil {
		t.Fatal("expected stale lease token to be rejected")
	}
	got, err := st.GetMissionRun(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "PAUSED" {
		t.Fatalf("stale worker overwrote state: %s", got.State)
	}
}
