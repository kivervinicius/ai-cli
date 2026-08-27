package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-manager/internal/profile"
)

func TestRunUsesIsolatedCodexHomeAndPreservesCWD(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	binDir := t.TempDir()
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "out.txt")
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_TEST_OUT", output)

	fake := filepath.Join(binDir, "codex")
	script := `#!/bin/sh
printf 'home=%s\n' "$CODEX_HOME" > "$AI_TEST_OUT"
printf 'cwd=%s\n' "$PWD" >> "$AI_TEST_OUT"
printf 'args=%s\n' "$*" >> "$AI_TEST_OUT"
`
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	if _, err := profile.Create("codex", "one"); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	if err := Run("one", []string{"--flag", "value"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	home, _ := profile.Home("codex", "one")
	if !strings.Contains(got, "home="+home) {
		t.Fatalf("wrong CODEX_HOME:\n%s", got)
	}
	if !strings.Contains(got, "cwd="+project) {
		t.Fatalf("cwd not preserved:\n%s", got)
	}
	if !strings.Contains(got, "args=--flag value") {
		t.Fatalf("args not preserved:\n%s", got)
	}
}

func TestEnsureConfigFilePreservesExistingSettings(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config.toml")

	initial := `# User custom configuration
model = "o3-mini"
temperature = 0.7

[features]
auto_update = false
`
	if err := os.WriteFile(cfg, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	if err := EnsureConfigFile(cfg); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	if !strings.Contains(content, `cli_auth_credentials_store = "file"`) {
		t.Fatalf("missing credentials store setting:\n%s", content)
	}
	if !strings.Contains(content, `model = "o3-mini"`) {
		t.Fatalf("lost existing model setting:\n%s", content)
	}
	if !strings.Contains(content, `temperature = 0.7`) {
		t.Fatalf("lost existing temperature setting:\n%s", content)
	}
	if !strings.Contains(content, `auto_update = false`) {
		t.Fatalf("lost existing features setting:\n%s", content)
	}

	// Calling again should be idempotent
	if err := EnsureConfigFile(cfg); err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(cfg)
	if string(raw2) != content {
		t.Fatalf("EnsureConfigFile not idempotent:\n%s\nvs\n%s", string(raw2), content)
	}
}

func TestCodexMultiProfileIsolation(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	binDir := t.TempDir()
	outputA := filepath.Join(t.TempDir(), "outA.txt")
	outputB := filepath.Join(t.TempDir(), "outB.txt")
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)

	fake := filepath.Join(binDir, "codex")
	script := `#!/bin/sh
if [ -n "$AI_TEST_OUT" ]; then
  printf 'home=%s\n' "$CODEX_HOME" > "$AI_TEST_OUT"
fi
`
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	if _, err := profile.Create("codex", "openai-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.Create("codex", "openai-b"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AI_TEST_OUT", outputA)
	if err := Run("openai-a", nil); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AI_TEST_OUT", outputB)
	if err := Run("openai-b", nil); err != nil {
		t.Fatal(err)
	}

	rawA, _ := os.ReadFile(outputA)
	rawB, _ := os.ReadFile(outputB)

	homeA, _ := profile.Home("codex", "openai-a")
	homeB, _ := profile.Home("codex", "openai-b")

	if !strings.Contains(string(rawA), "home="+homeA) {
		t.Fatalf("profile A wrong home: %s", string(rawA))
	}
	if !strings.Contains(string(rawB), "home="+homeB) {
		t.Fatalf("profile B wrong home: %s", string(rawB))
	}
	if homeA == homeB {
		t.Fatal("profiles should have distinct homes")
	}
}

func TestCodexSharedConversationsAndContext(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)

	// Create simulated host .codex with sessions and history
	hostCodex := filepath.Join(hostHome, ".codex")
	if err := os.MkdirAll(filepath.Join(hostCodex, "sessions"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostCodex, "history.jsonl"), []byte(`{"session":"123"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostCodex, "sessions", "123.json"), []byte(`{"id":"123"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostCodex, "auth.json"), []byte(`{"host_secret":"do_not_share"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := profile.Create("codex", "openai-a"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare("openai-a"); err != nil {
		t.Fatal(err)
	}

	homeA, _ := profile.Home("codex", "openai-a")

	// Verify sessions and history are accessible
	if _, err := os.Stat(filepath.Join(homeA, "sessions", "123.json")); err != nil {
		t.Fatalf("session file not accessible in profile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeA, "history.jsonl")); err != nil {
		t.Fatalf("history.jsonl not accessible in profile: %v", err)
	}

	// Verify auth.json is NOT linked to host auth.json
	profileAuth := filepath.Join(homeA, "auth.json")
	if _, err := os.Lstat(profileAuth); err == nil {
		t.Fatal("profile auth.json should not exist yet or be linked")
	}
}

func TestCodexMigratesExistingUnlinkedDir(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)

	hostCodex := filepath.Join(hostHome, ".codex")
	if err := os.MkdirAll(filepath.Join(hostCodex, "sessions"), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := profile.Create("codex", "migrate-codex"); err != nil {
		t.Fatal(err)
	}

	home, _ := profile.Home("codex", "migrate-codex")

	// Create a local session file in an existing unlinked directory
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "sessions", "session-old.json"), []byte(`{"id":"old"}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Prepare("migrate-codex"); err != nil {
		t.Fatal(err)
	}

	// Verify session-old.json was migrated to host
	if _, err := os.Stat(filepath.Join(hostCodex, "sessions", "session-old.json")); err != nil {
		t.Fatalf("session file was not migrated to host: %v", err)
	}

	// Verify home/sessions is now a symlink
	fi, err := os.Lstat(filepath.Join(home, "sessions"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("home/sessions should be a symlink now: %v", err)
	}
}
