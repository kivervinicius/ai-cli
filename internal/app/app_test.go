package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-manager/internal/profile"
)

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
	binDir = t.TempDir()
	testOut = filepath.Join(t.TempDir(), "test_out.txt")

	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_TEST_OUT", testOut)

	// Setup fake codex, agy, dbus-run-session, gnome-keyring-daemon
	writeExe := func(name, body string) {
		p := filepath.Join(binDir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeExe("codex", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=codex" > "$AI_TEST_OUT"
  echo "home=$CODEX_HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  echo "uid=$(id -u)" >> "$AI_TEST_OUT"
  echo "gid=$(id -g)" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)

	writeExe("agy", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=agy" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  echo "uid=$(id -u)" >> "$AI_TEST_OUT"
  echo "gid=$(id -g)" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	return binDir, testOut
}

func TestAppCLICommands(t *testing.T) {
	_, testOut := setupTestEnvironment(t)

	// Test version
	out, err := captureStdout(func() error {
		return Run([]string{"version"})
	})
	if err != nil || !strings.Contains(out, "ai-manager") {
		t.Fatalf("version failed: out=%q, err=%v", out, err)
	}

	// Test help
	out, err = captureStdout(func() error {
		return Run([]string{"help"})
	})
	if err != nil || !strings.Contains(out, "Usage:") {
		t.Fatalf("help failed: out=%q, err=%v", out, err)
	}

	// Test doctor
	out, err = captureStdout(func() error {
		return Run([]string{"doctor"})
	})
	if err != nil || !strings.Contains(out, "Distribution:") {
		t.Fatalf("doctor failed: out=%q, err=%v", out, err)
	}

	// Test paths
	out, err = captureStdout(func() error {
		return Run([]string{"paths"})
	})
	if err != nil || !strings.Contains(out, "data:") {
		t.Fatalf("paths failed: out=%q, err=%v", out, err)
	}

	// Test completion bash & zsh
	out, err = captureStdout(func() error {
		return Run([]string{"completion", "bash"})
	})
	if err != nil || !strings.Contains(out, "_ai_completion") {
		t.Fatalf("bash completion failed: out=%q, err=%v", out, err)
	}
	out, err = captureStdout(func() error {
		return Run([]string{"completion", "zsh"})
	})
	if err != nil || !strings.Contains(out, "#compdef ai") {
		t.Fatalf("zsh completion failed: out=%q, err=%v", out, err)
	}

	// Test add profiles
	if err := Run([]string{"add", "codex", "openai-1", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "codex", "openai-2", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "agy", "google-1", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "agy", "google-2", "--no-login"}); err != nil {
		t.Fatal(err)
	}

	// Test list
	out, err = captureStdout(func() error {
		return Run([]string{"list"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "openai-1") || !strings.Contains(out, "google-1") {
		t.Fatalf("list output missing profiles:\n%s", out)
	}

	// Test use / current
	if err := Run([]string{"use", "codex", "openai-2"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"use", "agy", "google-2"}); err != nil {
		t.Fatal(err)
	}

	out, err = captureStdout(func() error {
		return Run([]string{"current"})
	})
	if err != nil || !strings.Contains(out, "openai-2") || !strings.Contains(out, "google-2") {
		t.Fatalf("current failed: out=%q, err=%v", out, err)
	}

	// Test inspect
	out, err = captureStdout(func() error {
		return Run([]string{"inspect", "codex", "openai-2"})
	})
	if err != nil || !strings.Contains(out, "CODEX_HOME") || !strings.Contains(out, "Working Dir:") {
		t.Fatalf("inspect codex failed:\n%s", out)
	}
	out, err = captureStdout(func() error {
		return Run([]string{"inspect", "agy:google-2"})
	})
	if err != nil || !strings.Contains(out, "XDG_CONFIG_HOME") || !strings.Contains(out, "Process UID/GID:") {
		t.Fatalf("inspect agy failed:\n%s", out)
	}

	// Test execution via run command passing arguments after --
	workDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	_ = os.Chdir(workDir)
	defer os.Chdir(oldCwd)

	_ = os.Remove(testOut)
	if err := Run([]string{"run", "codex", "openai-1", "--", "--model", "gpt-4o", "-p", "hello world"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	home1, _ := profile.Home("codex", "openai-1")
	if !strings.Contains(got, "home="+home1) || !strings.Contains(got, "cwd="+workDir) || !strings.Contains(got, "arg=--model") || !strings.Contains(got, "arg=hello world") {
		t.Fatalf("run codex failed:\n%s", got)
	}

	// Test short form agy:google-1 without -- and with --yolo / -c conversation id
	_ = os.Remove(testOut)
	if err := Run([]string{"agy:google-1", "--yolo", "-c", "conv-12345", "--prompt", "explain code"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	agyHome1, _ := profile.Home("agy", "google-1")
	if !strings.Contains(got, "home="+agyHome1) || !strings.Contains(got, "arg=--yolo") || !strings.Contains(got, "arg=-c") || !strings.Contains(got, "arg=conv-12345") {
		t.Fatalf("agy:google-1 with flags failed:\n%s", got)
	}

	// Test ai agy google-1 --yolo -c conv-999
	_ = os.Remove(testOut)
	if err := Run([]string{"agy", "google-1", "--yolo", "-c", "conv-999"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+agyHome1) || !strings.Contains(got, "arg=--yolo") || !strings.Contains(got, "arg=conv-999") {
		t.Fatalf("ai agy google-1 with flags failed:\n%s", got)
	}

	// Test ai agy --yolo -c conv-default (using default profile google-2)
	_ = os.Remove(testOut)
	if err := Run([]string{"agy", "--yolo", "-c", "conv-default"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	agyHome2, _ := profile.Home("agy", "google-2")
	if !strings.Contains(got, "home="+agyHome2) || !strings.Contains(got, "arg=--yolo") || !strings.Contains(got, "arg=conv-default") {
		t.Fatalf("ai agy default with flags failed:\n%s", got)
	}

	// Test ai run agy google-1 --yolo -c conv-run
	_ = os.Remove(testOut)
	if err := Run([]string{"run", "agy", "google-1", "--yolo", "-c", "conv-run"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+agyHome1) || !strings.Contains(got, "arg=--yolo") || !strings.Contains(got, "arg=conv-run") {
		t.Fatalf("ai run agy google-1 with flags failed:\n%s", got)
	}

	// Test ai run agy --yolo (using default profile google-2)
	_ = os.Remove(testOut)
	if err := Run([]string{"run", "agy", "--yolo"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+agyHome2) || !strings.Contains(got, "arg=--yolo") {
		t.Fatalf("ai run agy default with flags failed:\n%s", got)
	}

	// Test ai run agy:google-1 --yolo
	_ = os.Remove(testOut)
	if err := Run([]string{"run", "agy:google-1", "--yolo"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+agyHome1) || !strings.Contains(got, "arg=--yolo") {
		t.Fatalf("ai run agy:google-1 with flags failed:\n%s", got)
	}

	// Test remove command with --yes
	if err := Run([]string{"remove", "codex", "openai-2", "--yes"}); err != nil {
		t.Fatal(err)
	}
	if profile.Exists("codex", "openai-2") {
		t.Fatal("openai-2 should have been deleted")
	}

	// Default should have been updated to remaining profile openai-1
	d, err := profile.Default("codex")
	if err != nil || d != "openai-1" {
		t.Fatalf("expected default to be openai-1, got %q (err=%v)", d, err)
	}

	// Test codex with --yolo
	_ = os.Remove(testOut)
	if err := Run([]string{"codex:openai-1", "--yolo"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+home1) || !strings.Contains(got, "arg=--yolo") {
		t.Fatalf("codex:openai-1 --yolo failed:\n%s", got)
	}

	// Test ai resume with explicit conversation and profile
	_ = os.Remove(testOut)
	if err := Run([]string{"resume", "conv-uuid-999", "agy:google-1"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got = string(raw)
	if !strings.Contains(got, "home="+agyHome1) || !strings.Contains(got, "arg=--conversation=conv-uuid-999") {
		t.Fatalf("ai resume agy failed:\n%s", got)
	}

	// Test ai list, status, and usage
	if err := Run([]string{"list"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"status"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"usage"}); err != nil {
		t.Fatal(err)
	}
}
