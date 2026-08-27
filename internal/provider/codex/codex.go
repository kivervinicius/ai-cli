package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-manager/internal/profile"
	"ai-manager/internal/runtime"
)

func Prepare(name string) error {
	home, err := profile.Home("codex", name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return err
	}
	cfg := filepath.Join(home, "config.toml")
	if err := EnsureConfigFile(cfg); err != nil {
		return err
	}

	realHome := findHostHome()
	if realHome != "" {
		linkHostDotfiles(home, realHome)
		hostCodex := filepath.Join(realHome, ".codex")
		if _, err := os.Stat(hostCodex); err == nil {
			linkHostCodex(home, hostCodex)
		}
	}

	return nil
}

func findHostHome() string {
	if v := os.Getenv("AI_REAL_HOME"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if st, err := os.Stat("/home/desenvolvedor"); err == nil && st.IsDir() {
		return "/home/desenvolvedor"
	}
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	if u := os.Getenv("USER"); u != "" {
		p := filepath.Join("/home", u)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

func linkHostDotfiles(home, realHome string) {
	if realHome == "" || home == "" || realHome == home {
		return
	}
	dotfiles := []string{
		".gitconfig",
		".git-credentials",
		".ssh",
		".npmrc",
		".npm",
		".yarnrc",
		".docker",
		".kube",
		".gnupg",
		".bashrc",
		".zshrc",
		".profile",
		".node_repl_history",
		".viminfo",
	}
	for _, name := range dotfiles {
		src := filepath.Join(realHome, name)
		dst := filepath.Join(home, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		fi, err := os.Lstat(dst)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dst)
				if err == nil && target == src {
					continue
				}
				_ = os.Remove(dst)
			} else {
				_ = os.Remove(dst)
			}
		}
		_ = os.Symlink(src, dst)
	}
}

func linkHostCodex(home, hostCodex string) {
	sharedItems := []string{
		"sessions",
		"history.jsonl",
		"session_index.jsonl",
		"logs_2.sqlite",
		"logs_2.sqlite-shm",
		"logs_2.sqlite-wal",
		"thread_history_1.sqlite",
		"memories",
		"memories_1.sqlite",
		"skills",
		"rules",
		"prompts",
		"plugins",
		"attachments",
		"agents",
		"queue_1.sqlite",
		"queue_1.sqlite-shm",
		"queue_1.sqlite-wal",
		"goals_1.sqlite",
		"state_5.sqlite",
	}
	for _, item := range sharedItems {
		src := filepath.Join(hostCodex, item)
		dst := filepath.Join(home, item)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		fi, err := os.Lstat(dst)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dst)
				if err == nil && target == src {
					continue
				}
				_ = os.Remove(dst)
			} else if fi.IsDir() {
				migrateDir(dst, src)
				_ = os.RemoveAll(dst)
			} else {
				_ = os.Remove(dst)
			}
		}
		_ = os.Symlink(src, dst)
	}
}

func migrateDir(srcDir, dstDir string) {
	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == srcDir {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return nil
		}
		targetPath := filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if data, err := os.ReadFile(path); err == nil {
				_ = os.WriteFile(targetPath, data, info.Mode().Perm())
			}
		}
		return nil
	})
}

func findHostCodexDir() string {
	candidates := []string{}
	if v := os.Getenv("AI_REAL_HOME"); v != "" {
		candidates = append(candidates, filepath.Join(v, ".codex"))
	}
	candidates = append(candidates, "/home/desenvolvedor/.codex")
	if h, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(h, ".codex"))
	}
	if u := os.Getenv("USER"); u != "" {
		candidates = append(candidates, filepath.Join("/home", u, ".codex"))
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

// EnsureConfigFile guarantees cli_auth_credentials_store = "file" while preserving existing content and 0600 permissions.
func EnsureConfigFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte("cli_auth_credentials_store = \"file\"\n"), 0600)
		}
		return err
	}

	lines := strings.Split(string(raw), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "cli_auth_credentials_store") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				lines[i] = "cli_auth_credentials_store = \"file\""
				found = true
				break
			}
		}
	}
	if !found {
		content := strings.TrimRight(string(raw), "\n")
		if content == "" {
			content = "cli_auth_credentials_store = \"file\""
		} else {
			content = content + "\ncli_auth_credentials_store = \"file\""
		}
		return os.WriteFile(path, []byte(content+"\n"), 0600)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

func Login(name string) error              { return run(name, []string{"login"}) }
func Run(name string, args []string) error { return run(name, args) }
func LoginStatus(name string) error        { return run(name, []string{"login", "status"}) }

func run(name string, args []string) error {
	if !profile.Exists("codex", name) {
		return fmt.Errorf("profile codex:%s does not exist", name)
	}
	if err := Prepare(name); err != nil {
		return err
	}
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return err
	}
	home, _ := profile.Home("codex", name)
	cwd, _ := os.Getwd()
	env := runtime.EnvSet(os.Environ(), map[string]string{"CODEX_HOME": home})
	return runtime.RunInteractive(bin, args, env, cwd)
}
