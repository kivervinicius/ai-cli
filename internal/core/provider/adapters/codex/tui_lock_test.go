package codex

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestTryAcquireTUILockNonBlocking(t *testing.T) {
	home := t.TempDir()
	held, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = held.Release() }()

	if !IsTUILocked(home) {
		t.Fatal("expected IsTUILocked=true while lock is held")
	}

	lock, ok, err := TryAcquireTUILock(home)
	if err != nil {
		t.Fatalf("TryAcquireTUILock: %v", err)
	}
	if ok || lock != nil {
		t.Fatal("expected non-blocking acquire to fail while TUI holds the lock")
	}
}

func TestRunAppServerRateLimitsRespectsTUILock(t *testing.T) {
	home := t.TempDir()
	lock, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var spawned atomic.Int32
	prev := appServerCommand
	t.Cleanup(func() { appServerCommand = prev })
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		spawned.Add(1)
		return prev(ctx, codexHome)
	}

	_, err = runAppServerRateLimits(context.Background(), home)
	if err == nil || !strings.Contains(err.Error(), "TUI") {
		t.Fatalf("expected TUI busy error, got %v", err)
	}
	if spawned.Load() != 0 {
		t.Fatalf("must not spawn app-server under TUI lock; spawned=%d", spawned.Load())
	}
}

func TestGetUsageSkipsMigrationAndAppServerWhileTUILocked(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CODEX_APP_SERVER", "1")

	profileName := "busy"
	home := filepath.Join(data, "profiles", "codex", profileName, "home")
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-busy", "busy@example.test")

	hostSessions := filepath.Join(t.TempDir(), "host-sessions")
	if err := os.MkdirAll(hostSessions, 0700); err != nil {
		t.Fatal(err)
	}
	threadPath := filepath.Join(home, "thread_history_1.sqlite")
	if err := os.Symlink(filepath.Join(hostSessions, "thread_history_1.sqlite"), threadPath); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	writeRollout(t, filepath.Join(home, "sessions", "rollout-busy.jsonl"), 12, 40, now.Add(time.Hour).Unix(), now, "")

	lock, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var spawned atomic.Int32
	prev := appServerCommand
	t.Cleanup(func() { appServerCommand = prev })
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		spawned.Add(1)
		return prev(ctx, codexHome)
	}

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: profileName})
	if spawned.Load() != 0 {
		t.Fatalf("app-server must not spawn while TUI lock is held; spawned=%d", spawned.Load())
	}
	if len(snap.Windows) == 0 {
		t.Fatalf("expected rollout windows while TUI locked, got %+v", snap)
	}
	if fi, err := os.Lstat(threadPath); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("GetUsage must not replace thread_history symlink while TUI is locked: err=%v mode=%v", err, fi)
	}
}

func TestPrepareUsesCanonicalHomeOnly(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "canon"}); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(data, "profiles", "codex", "canon", "home")
	if _, err := os.Stat(filepath.Join(home, "sessions")); err != nil {
		t.Fatalf("canonical sessions missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		t.Fatalf("canonical config missing: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(home, ".codex", "sessions")); err == nil && fi.IsDir() {
		t.Fatal("Prepare must not create a second .codex/sessions tree")
	}
}

func TestPrepareMigratesLegacyConfigBeforeCreatingCanonicalConfig(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	home := filepath.Join(data, "profiles", "codex", "legacy-config", "home")
	legacyConfig := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(legacyConfig), 0700); err != nil {
		t.Fatal(err)
	}
	const contents = "model = \"legacy-model\"\n"
	if err := os.WriteFile(legacyConfig, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "legacy-config"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read canonical config: %v", err)
	}
	if !strings.Contains(string(got), contents) {
		t.Fatalf("canonical config = %q, want migrated legacy contents containing %q", got, contents)
	}
}

func TestPrepareWaitsForActiveTUILock(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	home := filepath.Join(data, "profiles", "codex", "locked", "home")
	held, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}

	prepared := make(chan error, 1)
	go func() {
		prepared <- New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "locked"})
	}()

	select {
	case err := <-prepared:
		_ = held.Release()
		t.Fatalf("Prepare completed while TUI lock was held: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := held.Release(); err != nil {
		t.Fatalf("release held lock: %v", err)
	}

	select {
	case err := <-prepared:
		if err != nil {
			t.Fatalf("Prepare after lock release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Prepare did not continue after TUI lock release")
	}
}

func TestEnvCODEXHome(t *testing.T) {
	if got := EnvCODEXHome([]string{"PATH=/bin", "CODEX_HOME=/profiles/codex/work/home", "HOME=/profiles/codex/work/home"}); got != "/profiles/codex/work/home" {
		t.Fatalf("EnvCODEXHome=%q", got)
	}
	if got := EnvCODEXHome([]string{"HOME=/tmp"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
