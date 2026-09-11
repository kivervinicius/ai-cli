package contextsnapshot

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitTestCommand(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, output)
	}
}

func TestInspectCodeIdentityTracksWorktreeAndUntrackedChanges(t *testing.T) {
	root := t.TempDir()
	gitTestCommand(t, root, "init")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTestCommand(t, root, "add", "go.mod")
	gitTestCommand(t, root, "-c", "user.email=test@example.invalid", "-c", "user.name=Test", "commit", "-m", "initial")
	clean, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if clean.State != CodeIdentityClean || !clean.Complete || clean.HeadSHA == "" || clean.IdentityDigest == "" {
		t.Fatalf("unexpected clean identity: %+v", clean)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module changed\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if changed.State != CodeIdentityDirty || changed.WorktreeDigest == clean.WorktreeDigest || changed.IdentityDigest == clean.IdentityDigest {
		t.Fatalf("worktree change was not captured: clean=%+v changed=%+v", clean, changed)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("untracked"), 0600); err != nil {
		t.Fatal(err)
	}
	untracked, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if untracked.UntrackedDigest == "" || untracked.IdentityDigest == changed.IdentityDigest {
		t.Fatalf("untracked change was not captured: %+v", untracked)
	}
}

func TestDiscoverReturnsStructuredFactsWithProvenance(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo","scripts":{"test":"vitest"},"dependencies":{"react":"19"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.ts"), []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.test.ts"), []byte("test('ok',()=>{})"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: ci"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Discover(root, Metadata{ProjectID: "project-1"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProjectID != "project-1" || snapshot.ScannerVersion == "" || len(snapshot.Facts) < 4 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	for _, fact := range snapshot.Facts {
		if fact.ObservedAt.IsZero() || len(fact.Provenance) == 0 || strings.HasPrefix(fact.Provenance[0].SourcePath, root) {
			t.Fatalf("fact lacks safe provenance: %+v", fact)
		}
	}
}
