package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceStore_AddListRemove(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "projects.json")

	store := NewStore(storeFile)

	// Create test directories
	proj1 := filepath.Join(tempDir, "proj-alpha")
	proj2 := filepath.Join(tempDir, "proj-beta")
	_ = os.MkdirAll(proj1, 0755)
	_ = os.MkdirAll(proj2, 0755)

	p1, err := store.Add(proj1, "Alpha Project")
	if err != nil {
		t.Fatalf("failed to add proj1: %v", err)
	}
	if p1.Name != "Alpha Project" {
		t.Errorf("expected Alpha Project, got %s", p1.Name)
	}

	p2, err := store.Add(proj2, "")
	if err != nil {
		t.Fatalf("failed to add proj2: %v", err)
	}
	if p2.Name != "proj-beta" {
		t.Errorf("expected proj-beta default name, got %s", p2.Name)
	}

	list := store.List()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 projects, got %d", len(list))
	}

	// Remove proj1
	if err := store.Remove(proj1); err != nil {
		t.Fatalf("failed to remove proj1: %v", err)
	}

	// Verify reload from disk
	storeReloaded := NewStore(storeFile)
	list2 := storeReloaded.List()
	for _, p := range list2 {
		if p.Path == proj1 {
			t.Errorf("proj1 should have been removed from disk persistence")
		}
	}
}
