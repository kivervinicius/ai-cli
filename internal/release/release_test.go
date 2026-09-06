package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteRestoresVersionWhenFrontendFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("0.4.6\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(root)
	r.LocalBin = filepath.Join(root, "bin")
	r.Run = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("npm failed"), os.ErrPermission
	}
	_, err := r.Execute(context.Background(), "0.4.6", "0.4.7")
	if err == nil || !strings.Contains(err.Error(), "frontend build failed") {
		t.Fatalf("expected frontend failure, got %v", err)
	}
	b, readErr := os.ReadFile(filepath.Join(root, "VERSION"))
	if readErr != nil || string(b) != "0.4.6\n" {
		t.Fatalf("VERSION was not restored: %q, %v", b, readErr)
	}
}

func TestWithoutNodeOptionsRemovesPnPHook(t *testing.T) {
	got := withoutNodeOptions([]string{"PATH=/bin", "NODE_OPTIONS=--require .pnp.cjs", "HOME=/tmp"})
	if strings.Contains(strings.Join(got, "\n"), "NODE_OPTIONS=") {
		t.Fatalf("PnP hook was not removed: %v", got)
	}
}
