package launcher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestMain(m *testing.M) {
	testDir, err := os.MkdirTemp("", "ai-control-launcher-test-*")
	if err == nil {
		defer os.RemoveAll(testDir)
		_ = os.Setenv("AI_MANAGER_DATA_DIR", testDir)
		_ = os.Setenv("AI_CLI_DATA_DIR", testDir)
	}
	registry.ResetDefaultRegistryForTest()
	code := m.Run()
	registry.ResetDefaultRegistryForTest()
	os.Exit(code)
}

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
