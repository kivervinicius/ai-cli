package app

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/profile"
)

func TestMain(m *testing.M) {
	if os.Getenv("AI_FAKE_PROVIDER") == "1" {
		provider := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
		if out := os.Getenv("AI_TEST_OUT"); out != "" {
			var b strings.Builder
			b.WriteString("provider=")
			b.WriteString(provider)
			b.WriteByte('\n')
			if provider == "codex" {
				b.WriteString("home=")
				b.WriteString(os.Getenv("CODEX_HOME"))
			} else {
				b.WriteString("home=")
				b.WriteString(os.Getenv("HOME"))
			}
			b.WriteByte('\n')
			b.WriteString("cwd=")
			cwd, _ := os.Getwd()
			b.WriteString(cwd)
			for _, arg := range os.Args[1:] {
				b.WriteString("\narg=")
				b.WriteString(arg)
			}
			_ = os.WriteFile(out, []byte(b.String()+"\n"), 0644)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String(), err
}

func setupTestEnvironment(t *testing.T) (binDir, testOut string) {
	t.Helper()
	data := t.TempDir()
	cfg := t.TempDir()
	state := t.TempDir()
	binDir = t.TempDir()
	testOut = filepath.Join(t.TempDir(), "test_out.txt")

	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_STATE_DIR", state)
	t.Setenv("AI_TEST_OUT", testOut)

	writeExe := func(name, body string) {
		if runtime.GOOS == "windows" {
			data, err := os.ReadFile(os.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(binDir, name+".exe"), data, 0755); err != nil {
				t.Fatal(err)
			}
			return
		}
		p := filepath.Join(binDir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeExe("codex", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=codex" > "$AI_TEST_OUT"
  echo "home=$CODEX_HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("agy", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=agy" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("claude", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=claude" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("opencode", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=opencode" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("gemini", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=gemini" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)

	t.Setenv("PATH", binDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
	if runtime.GOOS == "windows" {
		t.Setenv("AI_FAKE_PROVIDER", "1")
	}
	return binDir, testOut
}

func TestControlPlaneCLICommands(t *testing.T) {
	_, testOut := setupTestEnvironment(t)

	// 1. Version
	out, err := captureStdout(func() error {
		return Run([]string{"version"})
	})
	if err != nil || !strings.Contains(out, "Nexus") {
		t.Fatalf("version failed: %s, %v", out, err)
	}

	// 2. Help
	out, err = captureStdout(func() error {
		return Run([]string{"help"})
	})
	if err != nil || !strings.Contains(out, "Usage:") {
		t.Fatalf("help failed: %s, %v", out, err)
	}

	// 3. Providers listing
	out, err = captureStdout(func() error {
		return Run([]string{"providers"})
	})
	if err != nil || !strings.Contains(out, "Codex") || !strings.Contains(out, "Claude Code") {
		t.Fatalf("providers failed: %s, %v", out, err)
	}

	// 4. Add profiles
	if err := Run([]string{"add", "codex", "work", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "codex", "personal", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "agy", "google-1", "--no-login"}); err != nil {
		t.Fatal(err)
	}

	// 5. Profiles listing
	out, err = captureStdout(func() error {
		return Run([]string{"profiles"})
	})
	if err != nil || !strings.Contains(out, "work") || !strings.Contains(out, "google-1") {
		t.Fatalf("profiles failed: %s, %v", out, err)
	}

	// 6. JSON output support
	out, err = captureStdout(func() error {
		return Run([]string{"providers", "--json"})
	})
	var providers map[string]struct {
		Name      string `json:"name"`
		Installed bool   `json:"installed"`
		Profiles  int    `json:"profiles"`
	}
	if err != nil || json.Unmarshal([]byte(out), &providers) != nil || len(providers) == 0 {
		t.Fatalf("providers --json failed: %s, %v", out, err)
	}
	for _, id := range []string{"codex", "claude"} {
		provider, ok := providers[id]
		if !ok || provider.Name == "" {
			t.Fatalf("providers --json omitted expected provider schema entries: %s", out)
		}
	}

	// 7. Inspect
	out, err = captureStdout(func() error {
		return Run([]string{"inspect", "codex:work"})
	})
	if err != nil || !strings.Contains(out, `"provider": "codex"`) {
		t.Fatalf("inspect failed: %s, %v", out, err)
	}

	// 8. Bindings
	if err := Run([]string{"bind", "codex:work"}); err != nil {
		t.Fatal(err)
	}
	out, err = captureStdout(func() error {
		return Run([]string{"bindings"})
	})
	if err != nil || !strings.Contains(out, "work") {
		t.Fatalf("bindings failed: %s, %v", out, err)
	}
	if err := Run([]string{"unbind", "codex"}); err != nil {
		t.Fatal(err)
	}

	// 9. Explain
	out, err = captureStdout(func() error {
		return Run([]string{"explain", "codex"})
	})
	if err != nil || !strings.Contains(out, "Smart Account Selection") {
		t.Fatalf("explain failed: %s, %v", out, err)
	}

	// 10. Run explicit profile
	_ = os.Remove(testOut)
	if err := Run([]string{"codex:personal", "--model", "gpt-5.6-sol", "-p", "hello"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	home, _ := profile.Home("codex", "personal")
	if !strings.Contains(got, "home="+home) || !strings.Contains(got, "arg=--model") {
		t.Fatalf("run explicit profile failed:\n%s", got)
	}

	// 11. Run resume with provider native syntax
	_ = os.Remove(testOut)
	if err := Run([]string{"resume", "session-uuid-1234", "codex:work"}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(testOut)
	got = string(raw)
	if !strings.Contains(got, "arg=resume") || !strings.Contains(got, "arg=session-uuid-1234") {
		t.Fatalf("resume codex failed (must use 'codex resume <id>'):\n%s", got)
	}

	// 12. Security command
	out, err = captureStdout(func() error {
		return Run([]string{"security"})
	})
	if err != nil || !strings.Contains(out, "Isolation Preset:") {
		t.Fatalf("security failed: %s, %v", out, err)
	}

	// 13. Doctor command
	out, err = captureStdout(func() error {
		return Run([]string{"doctor"})
	})
	if err != nil || !strings.Contains(out, "Nexus Diagnostics") {
		t.Fatalf("doctor failed: %s, %v", out, err)
	}

	// 14. Completion commands (bash, zsh, fish, powershell, pwsh)
	for _, sh := range []string{"bash", "zsh", "fish", "powershell", "pwsh"} {
		out, err = captureStdout(func() error {
			return Run([]string{"completion", sh})
		})
		if err != nil || len(out) == 0 {
			t.Fatalf("completion %s failed: %s, %v", sh, out, err)
		}
	}

	// 15. AI Control CLI commands
	out, err = captureStdout(func() error {
		return Run([]string{"control", "running", "--json"})
	})
	if err != nil || !strings.Contains(out, "[") {
		t.Fatalf("control running failed: %s, %v", out, err)
	}

	out, err = captureStdout(func() error {
		return Run([]string{"control", "cleanup"})
	})
	if err != nil || !strings.Contains(out, "Cleaned up") {
		t.Fatalf("control cleanup failed: %s, %v", out, err)
	}

	out, err = captureStdout(func() error {
		return Run([]string{"control", "doctor", "--json"})
	})
	if err != nil || !strings.Contains(out, "drivers") {
		t.Fatalf("control doctor failed: %s, %v", out, err)
	}
}

func TestPerformSystemUpdateDoesNotClaimNexusBinaryUpdated(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable := func(name, body string) {
		t.Helper()
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable("npm", "exit 0")
	writeExecutable("orquestrador-maestro", "if [ \"$1\" = \"version\" ]; then echo 0.1.0; fi")
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", t.TempDir())

	result := PerformSystemUpdate()
	if result.NexusUpdated {
		t.Fatal("update must not claim that the Nexus binary was updated when no binary update ran")
	}
	if !strings.Contains(result.Error, "Nexus binary update") {
		t.Fatalf("missing honest Nexus update status: %+v", result)
	}
}
