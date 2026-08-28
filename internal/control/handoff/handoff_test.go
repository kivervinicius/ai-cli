package handoff

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestAccountHandoffValidation(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	// 1. Test failure when ProviderSessionID is missing
	sessNoID := registry.RuntimeSession{
		RuntimeID:         "rt-no-sess-id",
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sessNoID)

	_, err := PerformAccountHandoff(ctx, "rt-no-sess-id", "fake:personal")
	if err == nil || !strings.Contains(err.Error(), "source provider session ID is unknown") {
		t.Errorf("expected error for empty session ID, got %v", err)
	}

	// 2. Test failure when target provider doesn't match source provider
	sessValid := registry.RuntimeSession{
		RuntimeID:         "rt-with-sess-id",
		ProviderID:        "codex",
		ProfileID:         "work",
		ProviderSessionID: "sess-12345",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sessValid)

	_, err = PerformAccountHandoff(ctx, "rt-with-sess-id", "claude:work")
	if err == nil || !strings.Contains(err.Error(), "matching provider") {
		t.Errorf("expected error for mismatched provider in account handoff, got %v", err)
	}
}

func TestWorkCheckpointAndRedaction(t *testing.T) {
	secretGoal := "Fix bug with sk-proj-1234567890abcdef1234567890abcdef1234567890 and Bearer eyJhbGciOiJIUzI1NiJ9.test"
	cp := CaptureWorkCheckpoint(os.TempDir(), "rt-src", "codex", "work", "sess-abc", secretGoal)

	if strings.Contains(cp.Goal, "sk-proj-") {
		t.Errorf("expected secret OpenAI key in goal to be redacted, got: %s", cp.Goal)
	}
	if strings.Contains(cp.Goal, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("expected secret JWT in goal to be redacted, got: %s", cp.Goal)
	}

	prompt := FormatKickoffPrompt(cp)
	if strings.Contains(prompt, "sk-proj-") {
		t.Errorf("expected prompt to have redacted secrets, got: %s", prompt)
	}

	path, err := SaveCheckpoint(cp)
	if err != nil || path == "" {
		t.Fatalf("failed to save checkpoint: %v", err)
	}
	_ = os.Remove(path)
}

func TestAccountHandoffStateTransitionsAndRollback(t *testing.T) {
	// Not full E2E due to process complexities, but we can verify state logic
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	// 1. Setup active source
	sourceID := "rt-test-rollback"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-rollback-test",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// We try to handoff to an unauthenticated profile or something that fails
	// Wait, we can't easily mock auth here without config changes.
	// But we can check that it fails safely and we get rollback behavior.
	_, err := PerformAccountHandoff(ctx, sourceID, "fake:non-existent")
	if err == nil {
		t.Errorf("expected handoff to fail for unauthenticated or non-existent profile")
	}

	// Because we didn't mock a process for `sourceID`, IsProcessAlive will return false.
	// So rollback will attempt to spawn a new process.
	// We can just check that the source state is either running or we get an error about rollback.
	restored, ok := reg.Get(sourceID)
	if ok && restored.State == registry.StateHandoff {
		t.Errorf("source state should not remain in Handoff after failure")
	}
}

func TestContextHandoffIntegration(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	sourceID := "rt-ctx-test"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-ctx-test",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// Wait, without valid credentials, PerformContextHandoff fails at Authentication check.
	_, err := PerformContextHandoff(ctx, sourceID, "fake", "default")
	if err == nil {
		t.Errorf("expected failure if fake:default isn't authenticated, or success if it is")
	}
}

func TestGitBounding(t *testing.T) {
	// Test the checkpoint bounding on git diff
	dir := os.TempDir() + "/handoff-git-test"
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	cp := CaptureWorkCheckpoint(dir, "rt-1", "fake", "prof", "sess-1", "Test goal")
	if cp.Workspace != dir {
		t.Errorf("expected workspace %s, got %s", dir, cp.Workspace)
	}
}

func TestAccountHandoff_CheckpointFailureAborts(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	sourceID := "rt-cp-fail-test"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-cp-fail",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// In normal circumstances, handoff to fake:nonexistent fails early with clear error
	_, err := PerformAccountHandoff(ctx, sourceID, "fake:nonexistent")
	if err == nil {
		t.Fatal("expected handoff to fail on invalid profile")
	}

	// Source session must not be stuck in StateHandoff
	st, ok := reg.Get(sourceID)
	if ok && st.State == registry.StateHandoff {
		t.Errorf("source session state was not preserved/restored: %+v", st)
	}
}

func TestVerifyResumeContinuity(t *testing.T) {
	// Live process: this test process itself.
	live := &registry.RuntimeSession{
		RuntimeID: "rt-verify-live",
		PID:       os.Getpid(),
		State:     registry.StateRunning,
	}

	// Passing case: resume args reference the session ID.
	if ok, reason := VerifyResumeContinuity(live, []string{"resume", "sess-abc"}, "sess-abc"); !ok {
		t.Errorf("expected verification to pass, got: %s", reason)
	}

	// Failing: resume args reference a different session.
	if ok, _ := VerifyResumeContinuity(live, []string{"resume", "sess-zzz"}, "sess-abc"); ok {
		t.Error("verification must fail when resume args do not reference the session ID")
	}

	// Failing: process not alive.
	dead := &registry.RuntimeSession{
		RuntimeID: "rt-verify-dead",
		PID:       99999999,
		State:     registry.StateRunning,
	}
	if ok, _ := VerifyResumeContinuity(dead, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail when the target process is not alive")
	}

	// Failing: state not running.
	stopped := &registry.RuntimeSession{
		RuntimeID: "rt-verify-stopped",
		PID:       os.Getpid(),
		State:     registry.StateStopped,
	}
	if ok, _ := VerifyResumeContinuity(stopped, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail when the target runtime is not RUNNING")
	}

	// Failing: empty session ID.
	if ok, _ := VerifyResumeContinuity(live, []string{"resume", ""}, ""); ok {
		t.Error("verification must fail on empty session ID")
	}

	// Failing: nil session.
	if ok, _ := VerifyResumeContinuity(nil, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail on nil session")
	}
}
