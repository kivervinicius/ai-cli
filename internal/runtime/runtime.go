package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// LookPath searches for an executable in the system PATH and standard developer directories
// (e.g. ~/.local/bin, ~/.bun/bin, ~/.opencode/bin, ~/.cargo/bin, ~/.nvm/versions/node/*/bin).
func LookPath(name string) (string, error) {
	// 1. Standard PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// 2. Proactive developer directories
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidates := []string{
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, ".bun", "bin", name),
			filepath.Join(home, ".opencode", "bin", name),
			filepath.Join(home, ".cargo", "bin", name),
			filepath.Join(home, ".local", "share", "pnpm", name),
			filepath.Join("/usr", "local", "bin", name),
			filepath.Join("/snap", "bin", name),
		}

		for _, cand := range candidates {
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
				return cand, nil
			}
		}

		// Check NVM directories
		nvmPattern := filepath.Join(home, ".nvm", "versions", "node", "*", "bin", name)
		if matches, err := filepath.Glob(nvmPattern); err == nil && len(matches) > 0 {
			for i := len(matches) - 1; i >= 0; i-- {
				cand := matches[i]
				if fi, err := os.Stat(cand); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
					return cand, nil
				}
			}
		}
	}

	return "", exec.ErrNotFound
}

// EnvSet applies environment overrides and removes unset keys from the base environment slice.
func EnvSet(base []string, overrides map[string]string, unset ...string) []string {
	unsetMap := make(map[string]bool)
	for _, k := range unset {
		unsetMap[k] = true
	}

	overrideKeys := make(map[string]bool)
	for k := range overrides {
		overrideKeys[k] = true
	}

	var out []string
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 0 {
			continue
		}
		k := parts[0]
		if unsetMap[k] || overrideKeys[k] {
			continue
		}
		out = append(out, entry)
	}

	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
}

// RunInteractive executes an external CLI in full interactive TTY passthrough mode.
func RunInteractive(bin string, args []string, env []string, cwd string) (model.Failure, error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin

	var errBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fail := classifier.Classify(err, errBuf.String())
			return fail, err
		}
		return model.Failure{Kind: model.FailureUnknown, Message: err.Error()}, err
	}

	return model.Failure{Kind: model.FailureNone}, nil
}

// RunCommandCapture executes a command non-interactively and captures its combined stdout and stderr.
func RunCommandCapture(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	cmd.Dir = cwd

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.String(), err
}

// InternalBinDir ensures helper binaries (such as ai-browser / xdg-open shim) exist in a runtime dir.
func InternalBinDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(dataDir, "runtime", "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		return "", err
	}

	selfExe, err := os.Executable()
	if err != nil {
		return binDir, nil
	}

	for _, name := range []string{"ai-browser", "xdg-open"} {
		dst := filepath.Join(binDir, name)
		_ = os.Remove(dst)
		_ = os.Symlink(selfExe, dst)
	}

	return binDir, nil
}
