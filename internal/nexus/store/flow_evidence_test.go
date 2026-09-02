package store

import (
	"path/filepath"
	"testing"
)

func TestFlowEvidenceSQLiteRoundTripAndStableIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(Project{Name: "Evidence", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Flow"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(MissionRunRecord{ID: "run-1", PlanID: plan.ID, ProjectID: project.ID, State: "EXECUTING", PayloadJSON: `{}`}); err != nil {
		t.Fatal(err)
	}

	capsule, err := st.UpsertFlowContextCapsule(FlowContextCapsuleRecord{ID: "capsule-1", RunID: "run-1", StepID: "A", FlowRevision: 1, ContentJSON: `{"id":"capsule-1"}`})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := st.UpsertFlowWorkReceipt(FlowWorkReceiptRecord{ID: "receipt-1", RunID: "run-1", StepID: "A", Status: "VERIFIED", ContentJSON: `{"id":"receipt-1"}`})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	loadedCapsule, err := st.GetFlowContextCapsule("run-1", "A")
	if err != nil {
		t.Fatal(err)
	}
	loadedReceipt, err := st.GetFlowWorkReceipt("run-1", "A")
	if err != nil {
		t.Fatal(err)
	}
	if loadedCapsule.ID != capsule.ID || loadedReceipt.ID != receipt.ID {
		t.Fatalf("evidence identity did not survive restart: capsule=%q receipt=%q", loadedCapsule.ID, loadedReceipt.ID)
	}

	updatedReceipt, err := st.UpsertFlowWorkReceipt(FlowWorkReceiptRecord{ID: "receipt-retry", RunID: "run-1", StepID: "A", Status: "VERIFIED", ContentJSON: `{"id":"receipt-1","commands":["go test"]}`})
	if err != nil {
		t.Fatal(err)
	}
	if updatedReceipt.ID != "receipt-1" {
		t.Fatalf("receipt identity changed on idempotent upsert: %q", updatedReceipt.ID)
	}
	list, err := st.ListFlowWorkReceipts("run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one receipt row, got %d", len(list))
	}
}
