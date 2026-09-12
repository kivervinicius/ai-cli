package nexus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestMissionValidationEvidenceBindsIdentityAndVerifiesChain(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "evidence", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-evidence", PlanID: "plan", PlanRevision: 3, ProjectID: project.ID, TotalIterations: 1}
	result := runner.VerificationResult{Command: "go test ./...", Passed: true, ExitCode: 0}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/test", result.Command, []runner.VerificationResult{result}, "OpenCode", "profile-a", "", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.VerifyValidationEvidenceChain(stream.ID); err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RepositoryState != "NON_GIT" || entries[0].IdentityDigest == "" || entries[0].Confidence != store.EvidenceObserved || entries[0].Outcome != store.EvidenceNotVerified {
		t.Fatalf("unexpected evidence entry: %+v", entries)
	}
	var evidence map[string]any
	if err := json.Unmarshal([]byte(entries[0].EvidenceJSON), &evidence); err != nil {
		t.Fatal(err)
	}
}

func TestStableEvidenceIDIsIdempotent(t *testing.T) {
	first := stableEvidenceID("run", "scenario", 1, 1)
	second := stableEvidenceID("run", "scenario", 1, 1)
	if first != second {
		t.Fatal("evidence id must be stable")
	}
	if first == stableEvidenceID("run", "scenario", 2, 1) {
		t.Fatal("different attempts must produce distinct evidence ids")
	}
}
