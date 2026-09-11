package driver

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/testutil"
)

func TestAGYBuildCommandDoesNotStartKeyringByDefault(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_AGY_ENABLE_SECRET_SERVICE", "")
	binDir := t.TempDir()
	bin := testutil.WriteFakeBinary(t, binDir, "agy", "")
	t.Setenv("PATH", binDir)

	gotBin, gotArgs, _, err := (NewAGYDriver()).BuildCommand(context.Background(), model.Profile{Name: "work", Provider: "agy"}, []string{"-p", "hello"})
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	if gotBin != bin {
		t.Fatalf("binary=%q, want direct AGY binary %q", gotBin, bin)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-p" {
		t.Fatalf("args=%v, want direct provider args", gotArgs)
	}
}
