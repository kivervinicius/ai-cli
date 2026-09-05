package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalWorkspacePath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "real_sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subDir: %v", err)
	}

	symlinkDir := filepath.Join(tempDir, "symlink_sub")
	if err := os.Symlink(subDir, symlinkDir); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// CanonicalWorkspacePath should resolve symlink to real_sub
	canonicalFromReal, err := CanonicalWorkspacePath(subDir)
	if err != nil {
		t.Fatalf("CanonicalWorkspacePath(subDir) failed: %v", err)
	}

	canonicalFromSymlink, err := CanonicalWorkspacePath(symlinkDir)
	if err != nil {
		t.Fatalf("CanonicalWorkspacePath(symlinkDir) failed: %v", err)
	}

	if canonicalFromReal != canonicalFromSymlink {
		t.Errorf("expected canonical paths to be identical, got %q vs %q", canonicalFromReal, canonicalFromSymlink)
	}

	// Test CanonicalExistingWorkspaceDir
	checkedDir, err := CanonicalExistingWorkspaceDir(symlinkDir)
	if err != nil {
		t.Fatalf("CanonicalExistingWorkspaceDir failed: %v", err)
	}
	if checkedDir != canonicalFromReal {
		t.Errorf("expected checked dir %q, got %q", canonicalFromReal, checkedDir)
	}
}
