package browser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Open(args []string) error {
	if len(args) == 0 {
		return errors.New("missing URL")
	}

	url := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			url = a
			break
		}
	}
	if url == "" {
		url = args[0]
	}

	// Test hook
	if out := os.Getenv("AI_TEST_BROWSER_OUT"); out != "" {
		return os.WriteFile(out, []byte(url+"\n"), 0600)
	}

	env := os.Environ()
	if host := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); host != "" {
		env = replace(env, "DBUS_SESSION_BUS_ADDRESS", host)
	}
	if home := os.Getenv("AI_REAL_HOME"); home != "" {
		env = replace(env, "HOME", home)
		env = unset(env, "XDG_CONFIG_HOME")
		env = unset(env, "XDG_CACHE_HOME")
		env = unset(env, "XDG_DATA_HOME")
		env = unset(env, "GNOME_KEYRING_CONTROL")
	}

	var candidates [][]string
	if custom := os.Getenv("AI_HOST_BROWSER"); custom != "" {
		candidates = append(candidates, []string{custom, url})
	}

	selfExe, _ := os.Executable()
	selfRealPath, _ := filepath.EvalSymlinks(selfExe)

	checkOpener := func(p string, extra ...string) {
		if realP, err := filepath.EvalSymlinks(p); err == nil {
			if realP != selfRealPath {
				c := append([]string{p}, extra...)
				c = append(c, url)
				candidates = append(candidates, c)
			}
		}
	}

	checkOpener("/usr/bin/xdg-open")
	checkOpener("/bin/xdg-open")
	checkOpener("/usr/bin/gio", "open")
	checkOpener("/usr/bin/sensible-browser")
	checkOpener("/usr/bin/google-chrome-stable")
	checkOpener("/usr/bin/google-chrome")
	checkOpener("/usr/bin/firefox")
	checkOpener("/usr/bin/chromium-browser")
	checkOpener("/usr/bin/chromium")

	var lastErr error
	for _, c := range candidates {
		if _, err := os.Stat(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("failed to open browser: %w", lastErr)
	}
	return errors.New("no host browser opener found (expected xdg-open, gio, or browser binary)")
}

func replace(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			continue
		}
		out = append(out, e)
	}
	return append(out, prefix+value)
}

func unset(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			continue
		}
		out = append(out, e)
	}
	return out
}


