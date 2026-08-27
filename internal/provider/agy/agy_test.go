package agy

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"ai-manager/internal/profile"
)

func TestRunUsesIsolatedHomePreservesCWDAndUID(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	binDir := t.TempDir()
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "out.txt")
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_TEST_OUT", output)

	writeExe := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)
	writeExe("agy", `{
  echo "home=$HOME"
  echo "cwd=$PWD"
  echo "uid=$(id -u)"
  echo "gid=$(id -g)"
  echo "xdg=$XDG_DATA_HOME"
  echo "args=$*"
} > "$AI_TEST_OUT"`)

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	if _, err := profile.Create("agy", "one"); err != nil {
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
	home, _ := profile.Home("agy", "one")
	if !strings.Contains(got, "home="+home) {
		t.Fatalf("wrong HOME:\n%s", got)
	}
	if !strings.Contains(got, "cwd="+project) {
		t.Fatalf("cwd not preserved:\n%s", got)
	}
	if !strings.Contains(got, "args=--flag value") {
		t.Fatalf("args not preserved:\n%s", got)
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "uid="+current.Uid) {
		t.Fatalf("UID changed:\n%s", got)
	}
	if !strings.Contains(got, "gid="+current.Gid) {
		t.Fatalf("GID changed:\n%s", got)
	}
}

func TestAgyMultiProfileIsolation(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)

	writeExe := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)
	writeExe("agy", `if [ -n "$AI_TEST_OUT" ]; then
  echo "home=$HOME" > "$AI_TEST_OUT"
  echo "xdg_data=$XDG_DATA_HOME" >> "$AI_TEST_OUT"
fi`)

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	profiles := []string{"google-a", "google-b", "google-c"}
	homes := make(map[string]string)

	for _, name := range profiles {
		if _, err := profile.Create("agy", name); err != nil {
			t.Fatal(err)
		}
		outPath := filepath.Join(t.TempDir(), name+".txt")
		t.Setenv("AI_TEST_OUT", outPath)
		if err := Run(name, nil); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		expectedHome, _ := profile.Home("agy", name)
		if !strings.Contains(string(raw), "home="+expectedHome) {
			t.Fatalf("profile %s: unexpected output: %s", name, string(raw))
		}
		homes[name] = expectedHome
	}

	// Verify all profiles have distinct homes
	if homes["google-a"] == homes["google-b"] || homes["google-b"] == homes["google-c"] || homes["google-a"] == homes["google-c"] {
		t.Fatalf("profiles share homes: %+v", homes)
	}
}

func TestAgySharedConversationsAndContext(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	binDir := t.TempDir()
	hostHome := t.TempDir()
	testOut := filepath.Join(t.TempDir(), "out.txt")

	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("AI_TEST_OUT", testOut)

	// Create simulated host .gemini conversation
	convDir := filepath.Join(hostHome, ".gemini", "antigravity-cli", "conversations")
	if err := os.MkdirAll(convDir, 0755); err != nil {
		t.Fatal(err)
	}
	testConvFile := filepath.Join(convDir, "15f92906-029d-4318-b01e-96c4a0bd23c4.db")
	if err := os.WriteFile(testConvFile, []byte("conversation-data"), 0644); err != nil {
		t.Fatal(err)
	}

	writeExe := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)
	writeExe("agy", `if [ -n "$AI_TEST_OUT" ]; then
  echo "home=$HOME" > "$AI_TEST_OUT"
  if [ -f "$HOME/.gemini/antigravity-cli/conversations/15f92906-029d-4318-b01e-96c4a0bd23c4.db" ]; then
    echo "found_conv=true" >> "$AI_TEST_OUT"
  else
    echo "found_conv=false" >> "$AI_TEST_OUT"
  fi
fi`)

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	if _, err := profile.Create("agy", "google-account-1"); err != nil {
		t.Fatal(err)
	}
	if err := Run("google-account-1", []string{"--conversation=15f92906-029d-4318-b01e-96c4a0bd23c4"}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "found_conv=true") {
		t.Fatalf("conversation not found in shared context:\n%s", got)
	}
}

func TestAgySharedUserHomeAndDotfiles(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	hostHome := t.TempDir()

	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)

	// Create host dotfiles and config subdirectories
	if err := os.WriteFile(filepath.Join(hostHome, ".gitconfig"), []byte("[user]\nname = Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hostHome, ".ssh"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostHome, ".ssh", "id_rsa"), []byte("dummy-key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hostHome, ".config", "gh"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostHome, ".config", "gh", "config.yml"), []byte("git_protocol: ssh"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := profile.Create("agy", "test-user-home"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare("test-user-home"); err != nil {
		t.Fatal(err)
	}

	home, _ := profile.Home("agy", "test-user-home")

	// Verify dotfiles and .config/gh are linked
	if _, err := os.Stat(filepath.Join(home, ".gitconfig")); err != nil {
		t.Fatalf(".gitconfig not linked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "id_rsa")); err != nil {
		t.Fatalf(".ssh not linked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "gh", "config.yml")); err != nil {
		t.Fatalf(".config/gh not linked: %v", err)
	}
}

func TestAgyMigratesExistingUnlinkedGeminiDir(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	hostHome := t.TempDir()

	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)

	// Ensure host .gemini exists
	hostGeminiConv := filepath.Join(hostHome, ".gemini", "antigravity-cli", "conversations")
	if err := os.MkdirAll(hostGeminiConv, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := profile.Create("agy", "migrate-test"); err != nil {
		t.Fatal(err)
	}

	home, _ := profile.Home("agy", "migrate-test")

	// Simulate an older existing regular directory created in profile home with a local conversation
	localConvDir := filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
	if err := os.MkdirAll(localConvDir, 0755); err != nil {
		t.Fatal(err)
	}
	localConvFile := filepath.Join(localConvDir, "local-conv-99.db")
	if err := os.WriteFile(localConvFile, []byte("local-conversation-data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run Prepare - should migrate local conversation to host and replace home/.gemini with symlink
	if err := Prepare("migrate-test"); err != nil {
		t.Fatal(err)
	}

	// Verify local-conv-99.db was migrated to host .gemini
	if _, err := os.Stat(filepath.Join(hostGeminiConv, "local-conv-99.db")); err != nil {
		t.Fatalf("local conversation was not migrated to host .gemini: %v", err)
	}

	// Verify home/.gemini is a directory with shared conversations symlink
	convLink := filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
	fi, err := os.Lstat(convLink)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("home/.gemini/antigravity-cli/conversations should be a symlink: %v", err)
	}
}

func TestAgyMultiProfileOAuthTokenIsolation(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	hostHome := t.TempDir()

	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_REAL_HOME", hostHome)

	// Create host .gemini with shared conversation
	convDir := filepath.Join(hostHome, ".gemini", "antigravity-cli", "conversations")
	if err := os.MkdirAll(convDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(convDir, "shared-conv-1.db"), []byte("shared-db"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Profile A
	if _, err := profile.Create("agy", "google-account-1"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare("google-account-1"); err != nil {
		t.Fatal(err)
	}
	homeA, _ := profile.Home("agy", "google-account-1")
	tokenA := filepath.Join(homeA, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if err := os.WriteFile(tokenA, []byte("token-for-account-1"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create Profile B
	if _, err := profile.Create("agy", "google-account-2"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare("google-account-2"); err != nil {
		t.Fatal(err)
	}
	homeB, _ := profile.Home("agy", "google-account-2")
	tokenB := filepath.Join(homeB, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if err := os.WriteFile(tokenB, []byte("token-for-account-2"), 0600); err != nil {
		t.Fatal(err)
	}

	// Re-run Prepare on both to ensure idempotency and no clobbering
	if err := Prepare("google-account-1"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare("google-account-2"); err != nil {
		t.Fatal(err)
	}

	// Verify Profile A token remained unchanged
	rawA, err := os.ReadFile(tokenA)
	if err != nil || string(rawA) != "token-for-account-1" {
		t.Fatalf("Profile A token clobbered: %s (err=%v)", string(rawA), err)
	}

	// Verify Profile B token remained unchanged
	rawB, err := os.ReadFile(tokenB)
	if err != nil || string(rawB) != "token-for-account-2" {
		t.Fatalf("Profile B token clobbered: %s (err=%v)", string(rawB), err)
	}

	// Verify BOTH profiles have access to the shared conversation
	if _, err := os.Stat(filepath.Join(homeA, ".gemini", "antigravity-cli", "conversations", "shared-conv-1.db")); err != nil {
		t.Fatalf("Profile A cannot see shared conversation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeB, ".gemini", "antigravity-cli", "conversations", "shared-conv-1.db")); err != nil {
		t.Fatalf("Profile B cannot see shared conversation: %v", err)
	}
}
