package launcher

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRuntimeLauncher_StandaloneLaunchAndHandshake(t *testing.T) {
	l := NewLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runtimeID := "rt-launcher-test"
	opts := LaunchOptions{
		RuntimeID:  runtimeID,
		ProviderID: "fake",
		ProfileID:  "default",
		Workspace:  os.TempDir(),
		Standalone: true,
		Timeout:    3 * time.Second,
	}

	sess, err := l.Launch(ctx, opts)
	if err != nil {
		t.Fatalf("launcher failed: %v", err)
	}

	if sess.RuntimeID != runtimeID {
		t.Errorf("expected runtime ID %s, got %s", runtimeID, sess.RuntimeID)
	}
	if sess.State != "RUNNING" {
		t.Errorf("expected state RUNNING, got %s", sess.State)
	}
}
