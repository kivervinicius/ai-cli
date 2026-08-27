package profile

import (
	"path/filepath"
	"testing"
)

func TestCreateListDefaultDelete(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	if _, err := Create("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("agy", "google-a"); err != nil {
		t.Fatal(err)
	}
	if !Exists("codex", "alpha") {
		t.Fatal("codex profile should exist")
	}
	if err := SetDefault("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	d, err := Default("codex")
	if err != nil {
		t.Fatal(err)
	}
	if d != "alpha" {
		t.Fatalf("default=%q", d)
	}
	ps, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("got %d profiles, want 2", len(ps))
	}
	if err := Delete("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	if Exists("codex", "alpha") {
		t.Fatal("profile should be deleted")
	}
}

func TestEnsureRandomSecretPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := EnsureRandomSecret(path); err != nil {
		t.Fatal(err)
	}
}
