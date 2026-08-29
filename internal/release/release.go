package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	Root     string
	LocalBin string
	Run      func(context.Context, string, ...string) ([]byte, error)
	Now      func() time.Time
}

type Result struct{ Version, BinaryPath, Frontend, GoBuild, Validation string }

func NewRunner(root string) Runner {
	home, _ := os.UserHomeDir()
	bin := os.Getenv("LOCAL_BIN")
	if bin == "" {
		bin = filepath.Join(home, ".local", "bin")
	}
	return Runner{Root: root, LocalBin: bin, Run: runCommand, Now: time.Now}
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if name == "npm" || name == "node" {
		cmd.Env = withoutNodeOptions(os.Environ())
	}
	return cmd.CombinedOutput()
}

func withoutNodeOptions(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, value := range env {
		if !strings.HasPrefix(value, "NODE_OPTIONS=") {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func (r Runner) ReadVersion() (string, error) {
	b, err := os.ReadFile(filepath.Join(r.Root, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read VERSION: %w", err)
	}
	v := strings.TrimSpace(string(b))
	if err := Validate(v); err != nil {
		return "", err
	}
	return v, nil
}

func (r Runner) Execute(ctx context.Context, old, next string) (Result, error) {
	if err := Validate(old); err != nil {
		return Result{}, fmt.Errorf("current version: %w", err)
	}
	if err := Validate(next); err != nil {
		return Result{}, fmt.Errorf("new version: %w", err)
	}
	versionPath := filepath.Join(r.Root, "VERSION")
	if err := os.WriteFile(versionPath, []byte(next+"\n"), 0644); err != nil {
		return Result{}, fmt.Errorf("write VERSION: %w", err)
	}
	restore := true
	defer func() {
		if restore {
			_ = os.WriteFile(versionPath, []byte(old+"\n"), 0644)
		}
	}()

	result := Result{Version: next, Frontend: "pending", GoBuild: "pending"}
	if out, err := r.Run(ctx, "npm", "--prefix", filepath.Join(r.Root, "web"), "run", "build"); err != nil {
		return result, fmt.Errorf("frontend build failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	result.Frontend = "ok"
	commit := "unknown"
	if out, err := r.Run(ctx, "git", "-C", r.Root, "rev-parse", "--short", "HEAD"); err == nil {
		commit = strings.TrimSpace(string(out))
	}
	ldflags := fmt.Sprintf("-s -w -X github.com/kivervinicius/ai-cli/internal/buildinfo.Version=%s -X github.com/kivervinicius/ai-cli/internal/buildinfo.Commit=%s -X github.com/kivervinicius/ai-cli/internal/buildinfo.BuildDate=%s", next, commit, r.Now().UTC().Format(time.RFC3339))
	bin := filepath.Join(r.Root, "nexus")
	if out, err := r.Run(ctx, "go", "build", "-ldflags="+ldflags, "-o", bin, "./cmd/nexus"); err != nil {
		return result, fmt.Errorf("go build failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	result.GoBuild = "ok"
	if err := os.MkdirAll(r.LocalBin, 0755); err != nil {
		return result, fmt.Errorf("create local bin: %w", err)
	}
	dest := filepath.Join(r.LocalBin, "nexus")
	if err := copyFile(bin, dest); err != nil {
		return result, fmt.Errorf("install nexus: %w", err)
	}
	alias := filepath.Join(r.LocalBin, "ai")
	_ = os.Remove(alias)
	if err := os.Symlink(dest, alias); err != nil {
		return result, fmt.Errorf("install ai alias: %w", err)
	}
	result.BinaryPath = dest
	out, err := r.Run(ctx, dest, "version", "--json")
	if err != nil {
		return result, fmt.Errorf("installed binary validation failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &info); err != nil || info.Version != next {
		return result, fmt.Errorf("installed binary reported version %q, want %q", info.Version, next)
	}
	result.Validation = "ok"
	restore = false
	return result, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	// Write to temporary file in the destination directory to allow atomic rename
	tmpDst := fmt.Sprintf("%s.tmp-%d", dst, time.Now().UnixNano())
	out, err := os.OpenFile(tmpDst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, st.Mode()|0100)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmpDst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpDst)
		return err
	}
	// Atomic rename unlinks the running inode and replaces it seamlessly without ETXTBSY ("text file busy")
	if err := os.Rename(tmpDst, dst); err != nil {
		_ = os.Remove(tmpDst)
		return err
	}
	return nil
}
