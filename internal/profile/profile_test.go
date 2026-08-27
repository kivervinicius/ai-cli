package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateListDefaultDelete(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)

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
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0600 {
		t.Fatalf("mode=%o, want 600", st.Mode().Perm())
	}
	first, _ := os.ReadFile(path)
	if err := EnsureRandomSecret(path); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("existing secret was unexpectedly replaced")
	}
}

func TestValidateName(t *testing.T) {
	good := []string{"a", "google-a", "openai_work", "one.two", "A1"}
	for _, s := range good {
		if err := ValidateName(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	bad := []string{"", "../x", "a/b", "has space", ".."}
	for _, s := range bad {
		if err := ValidateName(s); err == nil {
			t.Errorf("%q should fail", s)
		}
	}
}

func TestDeleteReassignsDefault(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)

	if _, err := Create("agy", "google-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("agy", "google-b"); err != nil {
		t.Fatal(err)
	}
	if err := SetDefault("agy", "google-a"); err != nil {
		t.Fatal(err)
	}

	// Delete default profile google-a
	if err := Delete("agy", "google-a"); err != nil {
		t.Fatal(err)
	}

	// Default should now be google-b
	d, err := Default("agy")
	if err != nil {
		t.Fatal(err)
	}
	if d != "google-b" {
		t.Fatalf("expected default to be google-b, got %q", d)
	}

	// google-b data should be completely intact
	if !Exists("agy", "google-b") {
		t.Fatal("google-b should still exist")
	}

	// Delete remaining profile
	if err := Delete("agy", "google-b"); err != nil {
		t.Fatal(err)
	}
	d2, _ := Default("agy")
	if d2 != "" {
		t.Fatalf("expected empty default, got %q", d2)
	}
}

func TestInspectMetadata(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)

	if _, err := Create("codex", "test-codex"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("agy", "test-agy"); err != nil {
		t.Fatal(err)
	}

	// Inspect codex
	infoCodex, err := Inspect("codex", "test-codex")
	if err != nil {
		t.Fatal(err)
	}
	if infoCodex.Profile.Name != "test-codex" || infoCodex.Profile.Provider != "codex" {
		t.Fatalf("wrong profile metadata: %+v", infoCodex.Profile)
	}
	if infoCodex.RootPath == "" || infoCodex.HomePath == "" {
		t.Fatal("expected non-empty paths")
	}
	if infoCodex.IsolationVars["CODEX_HOME"] != infoCodex.HomePath {
		t.Fatalf("expected CODEX_HOME=%s, got %s", infoCodex.HomePath, infoCodex.IsolationVars["CODEX_HOME"])
	}

	// Inspect agy
	infoAgy, err := Inspect("agy", "test-agy")
	if err != nil {
		t.Fatal(err)
	}
	if infoAgy.Profile.Name != "test-agy" || infoAgy.Profile.Provider != "agy" {
		t.Fatalf("wrong profile metadata: %+v", infoAgy.Profile)
	}
	if infoAgy.IsolationVars["HOME"] != infoAgy.HomePath {
		t.Fatalf("expected HOME=%s, got %s", infoAgy.HomePath, infoAgy.IsolationVars["HOME"])
	}
}
