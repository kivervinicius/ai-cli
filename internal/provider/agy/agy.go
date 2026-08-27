package agy

import (
	"fmt"
	"os"
	"path/filepath"

	"ai-manager/internal/profile"
	"ai-manager/internal/runtime"
)

func Prepare(name string) error {
	root, err := profile.Root("agy", name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	dirs := []string{
		home,
		filepath.Join(home, ".config"),
		filepath.Join(home, ".cache"),
		filepath.Join(home, ".local", "share"),
		filepath.Join(home, ".local", "share", "keyrings"),
		filepath.Join(root, "runtime", "keyring-control"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
		_ = os.Chmod(d, 0700)
	}
	if err := profile.EnsureRandomSecret(filepath.Join(root, "keyring.pass")); err != nil {
		return err
	}

	// Link host developer configurations, dotfiles, and AI spaces
	realHome := findHostHome()
	if realHome != "" {
		linkHostHome(home, realHome)
	}

	// Ensure helper binaries exist
	_, _ = runtime.InternalBinDir()
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

func linkHostHome(home, realHome string) {
	if realHome == "" || home == "" || realHome == home {
		return
	}

	linkItem := func(name string) {
		src := filepath.Join(realHome, name)
		dst := filepath.Join(home, name)
		if _, err := os.Stat(src); err != nil {
			return
		}
		fi, err := os.Lstat(dst)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dst)
				if err == nil && target == src {
					return
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

	// 1. Granular AI Spaces (conversations & context shared, tokens & auth isolated)
	linkHostGemini(home, realHome)
	linkItem(".cursor")
	linkItem(".vscode")
	linkItem(".opencode")

	// 2. Developer Configurations & Dotfiles
	linkItem(".gitconfig")
	linkItem(".git-credentials")
	linkItem(".ssh")
	linkItem(".npmrc")
	linkItem(".npm")
	linkItem(".yarnrc")
	linkItem(".docker")
	linkItem(".kube")
	linkItem(".gnupg")
	linkItem(".bashrc")
	linkItem(".zshrc")
	linkItem(".profile")
	linkItem(".node_repl_history")
	linkItem(".viminfo")

	// 3. User .config subdirectories
	hostConfig := filepath.Join(realHome, ".config")
	profileConfig := filepath.Join(home, ".config")
	if entries, err := os.ReadDir(hostConfig); err == nil {
		for _, e := range entries {
			if e.Name() == "ai-manager" {
				continue
			}
			src := filepath.Join(hostConfig, e.Name())
			dst := filepath.Join(profileConfig, e.Name())
			if fi, err := os.Lstat(dst); err == nil {
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
}

func linkHostGemini(home, realHome string) {
	hostGemini := filepath.Join(realHome, ".gemini")
	if _, err := os.Stat(hostGemini); err != nil {
		return
	}

	profileGemini := filepath.Join(home, ".gemini")
	if fi, err := os.Lstat(profileGemini); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(profileGemini)
		}
	}
	_ = os.MkdirAll(profileGemini, 0700)

	profileAgyCli := filepath.Join(profileGemini, "antigravity-cli")
	if fi, err := os.Lstat(profileAgyCli); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(profileAgyCli)
		}
	}
	_ = os.MkdirAll(profileAgyCli, 0700)

	linkItem := func(src, dst string) {
		if _, err := os.Stat(src); err != nil {
			return
		}
		fi, err := os.Lstat(dst)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dst)
				if err == nil && target == src {
					return
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

	// Shared items in .gemini/antigravity-cli/
	hostAgyCli := filepath.Join(hostGemini, "antigravity-cli")
	sharedCliItems := []string{
		"conversations",
		"brain",
		"history.jsonl",
		"conversation_summaries.db",
		"cache",
		"knowledge",
		"builtin",
		"bin",
	}
	for _, item := range sharedCliItems {
		linkItem(filepath.Join(hostAgyCli, item), filepath.Join(profileAgyCli, item))
	}

	// Shared items in .gemini/
	sharedGeminiItems := []string{
		"skills",
		"config",
		"extensions",
		"policies",
		"rules.toml",
		"GEMINI.md",
	}
	for _, item := range sharedGeminiItems {
		linkItem(filepath.Join(hostGemini, item), filepath.Join(profileGemini, item))
	}

	// Note: antigravity-oauth-token, oauth_creds.json, google_account_id,
	// google_accounts.json, jetski_state.pbtxt, and antigravity-browser-profile/
	// are NOT linked and remain 100% isolated per profile in profileGemini.
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

func Login(name string) error {
	// AGY has no standalone `login` command. On a fresh isolated profile,
	// launching the official CLI triggers Google Sign-In.
	return run(name, nil)
}

func Run(name string, args []string) error { return run(name, args) }

func run(name string, args []string) error {
	if !profile.Exists("agy", name) {
		return fmt.Errorf("profile agy:%s does not exist", name)
	}
	if err := Prepare(name); err != nil {
		return err
	}
	agyBin, err := runtime.LookPath("agy")
	if err != nil {
		return err
	}
	dbusRun, err := runtime.LookPath("dbus-run-session")
	if err != nil {
		return err
	}
	if _, err := runtime.LookPath("gnome-keyring-daemon"); err != nil {
		return err
	}

	root, _ := profile.Root("agy", name)
	home := filepath.Join(root, "home")
	passFile := filepath.Join(root, "keyring.pass")
	control := filepath.Join(root, "runtime", "keyring-control")
	internalBin, _ := runtime.InternalBinDir()
	realHome := findHostHome()
	cwd, _ := os.Getwd()

	hostDBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	hostPath := os.Getenv("PATH")
	envMap := map[string]string{
		"HOME":                             home,
		"XDG_CONFIG_HOME":                  filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":                   filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":                    filepath.Join(home, ".local", "share"),
		"AI_REAL_HOME":                     realHome,
		"AI_HOST_DBUS_SESSION_BUS_ADDRESS": hostDBus,
		"AI_KEYRING_PASSWORD_FILE":         passFile,
		"AI_KEYRING_CONTROL_DIR":           control,
		"AI_AGY_BIN":                       agyBin,
		"BROWSER":                          filepath.Join(internalBin, "ai-browser"),
		"PATH":                             internalBin + ":" + hostPath,
		"GIT_CONFIG_GLOBAL":                filepath.Join(realHome, ".gitconfig"),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}
	if _, err := os.Stat(filepath.Join(realHome, ".npmrc")); err == nil {
		envMap["NPM_CONFIG_USERCONFIG"] = filepath.Join(realHome, ".npmrc")
	}

	env := runtime.EnvSet(os.Environ(), envMap, "GNOME_KEYRING_CONTROL")

	// The shell only bootstraps an isolated Secret Service, then execs the
	// official agy binary in the current project directory. It never reads or
	// edits AGY OAuth tokens.
	script := `set -e
umask 077
rm -rf "$AI_KEYRING_CONTROL_DIR"
mkdir -p "$AI_KEYRING_CONTROL_DIR"
chmod 700 "$AI_KEYRING_CONTROL_DIR"
mkdir -p "$XDG_DATA_HOME/keyrings"
chmod 700 "$XDG_DATA_HOME/keyrings"
# Start/unlock a per-profile Secret Service keyring. Password never leaves the
# profile directory and is not an AGY/OpenAI credential.
eval "$(cat "$AI_KEYRING_PASSWORD_FILE" | gnome-keyring-daemon --unlock --components=secrets --control-directory="$AI_KEYRING_CONTROL_DIR")"
unset AI_KEYRING_PASSWORD_FILE
exec "$AI_AGY_BIN" "$@"
`
	runArgs := []string{"--", "bash", "-c", script, "ai-agy"}
	runArgs = append(runArgs, args...)
	return runtime.RunInteractive(dbusRun, runArgs, env, cwd)
}
