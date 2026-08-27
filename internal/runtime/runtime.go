package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func LookPath(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH", name)
	}
	return p, nil
}

func EnvSet(env []string, kv map[string]string, unset ...string) []string {
	drop := map[string]bool{}
	for k := range kv {
		drop[k] = true
	}
	for _, k := range unset {
		drop[k] = true
	}
	out := make([]string, 0, len(env)+len(kv))
	for _, e := range env {
		k := e
		if i := strings.IndexByte(e, '='); i >= 0 {
			k = e[:i]
		}
		if !drop[k] {
			out = append(out, e)
		}
	}
	for k, v := range kv {
		out = append(out, k+"="+v)
	}
	return out
}

// RunInteractive executes a command connected to standard I/O with signal propagation.
func RunInteractive(path string, args []string, env []string, dir string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 8)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

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
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if ws.Signaled() {
					return err
				}
			}
		}
		return err
	}
	return nil
}

// InternalBinDir returns the directory containing helper symlinks (ai-browser, xdg-open).
// It ensures that the helper directory and valid symlinks exist pointing to the current executable.
func InternalBinDir() (string, error) {
	if custom := os.Getenv("AI_MANAGER_LIB_DIR"); custom != "" {
		d := filepath.Join(custom, "bin")
		_ = ensureSymlinks(d)
		return d, nil
	}

	home, err := os.UserHomeDir()
	if err == nil {
		d := filepath.Join(home, ".local", "lib", "ai-manager", "bin")
		if err := ensureSymlinks(d); err == nil {
			return d, nil
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	_ = ensureSymlinks(dir)
	return dir, nil
}

func ensureSymlinks(binDir string) error {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	for _, helper := range []string{"ai-browser", "xdg-open"} {
		target := filepath.Join(binDir, helper)
		if cur, err := os.Readlink(target); err == nil && cur == exe {
			continue
		}
		_ = os.Remove(target)
		_ = os.Symlink(exe, target)
	}
	return nil
}
