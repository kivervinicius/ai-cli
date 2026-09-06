package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryRunRepositoryRejectsLiveLeaseReacquireBySameOwner(t *testing.T) {
	repo := NewMemoryRunRepository()
	run := &MissionRun{ID: "run_same_owner", PlanID: "plan", ProjectID: "project", State: StateExecuting, StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	first, err := repo.AcquireLease(context.Background(), run.ID, "worker-a", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if first.LeaseToken == "" {
		t.Fatal("expected first fencing token")
	}
	if _, err := repo.AcquireLease(context.Background(), run.ID, "worker-a", time.Second); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("expected live same-owner lease reacquire to be rejected, got %v", err)
	}
}
