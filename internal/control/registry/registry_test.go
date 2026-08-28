package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryPersistenceAndLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")

	reg := NewRegistry(dbPath)

	sess := RuntimeSession{
		RuntimeID:    "rt-001",
		ProviderID:   "codex",
		ProfileID:    "work",
		Workspace:    "/projects/omega",
		PID:          os.Getpid(),
		State:        StateRunning,
		ControlLevel: ControlLevelAPI,
		StartedAt:    time.Now(),
	}

	if err := reg.Register(sess); err != nil {
		t.Fatalf("failed to register session: %v", err)
	}

	// Verify in-memory retrieval
	got, ok := reg.Get("rt-001")
	if !ok || got.ProviderID != "codex" || got.State != StateRunning {
		t.Fatalf("unexpected retrieved session: %+v", got)
	}

	// Verify persistence reload
	reg2 := NewRegistry(dbPath)
	got2, ok2 := reg2.Get("rt-001")
	if !ok2 || got2.ProfileID != "work" {
		t.Fatalf("failed to reload persisted session: %+v", got2)
	}

	// Update state
	if err := reg2.UpdateState("rt-001", StateStopped); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}
	if got3, _ := reg2.Get("rt-001"); got3.State != StateStopped {
		t.Errorf("expected state STOPPED, got %s", got3.State)
	}

	// Active list check
	if len(reg2.ListActive()) != 0 {
		t.Errorf("expected 0 active sessions after stopping")
	}

	// Delete
	if err := reg2.Delete("rt-001"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}
	if _, ok := reg2.Get("rt-001"); ok {
		t.Errorf("expected session to be deleted")
	}
}

func TestCleanupStale(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")
	reg := NewRegistry(dbPath)

	// Register a session with a non-existent PID
	fakePID := 99999999
	_ = reg.Register(RuntimeSession{
		RuntimeID: "rt-dead",
		PID:       fakePID,
		State:     StateRunning,
	})

	cleaned, err := reg.CleanupStale()
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned session, got %d", cleaned)
	}

	s, _ := reg.Get("rt-dead")
	if s.State != StateStale {
		t.Errorf("expected state STALE, got %s", s.State)
	}
}
