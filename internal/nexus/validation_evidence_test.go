package nexus

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func TestMissionValidationEvidenceRecordsFailedResultsAsFail(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	// Initialize a git repo so HeadSHA is present and FAIL can be recorded.
	git := exec.Command("git", "init")
	git.Dir = root
	if out, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"add", "."},
		{"commit", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	project, err := st.CreateProject(store.Project{Name: "fail-evidence", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-fail", PlanID: "plan", PlanRevision: 1, ProjectID: project.ID}
	result := runner.VerificationResult{Command: "go test ./...", Passed: false, ExitCode: 1}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/test", result.Command, []runner.VerificationResult{result}, "OpenCode", "profile-a", "", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Outcome != store.EvidenceFail || entries[0].Confidence != store.EvidenceObserved {
		t.Fatalf("failed verification must record FAIL: %+v", entries)
	}
}
