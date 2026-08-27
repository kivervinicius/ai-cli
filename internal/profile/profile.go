package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Profile struct {
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Config struct {
	Defaults map[string]string `json:"defaults"`
}

func DataDir() (string, error) {
	if v := os.Getenv("AI_MANAGER_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "ai-manager"), nil
}

func ConfigDir() (string, error) {
	if v := os.Getenv("AI_MANAGER_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ai-manager"), nil
}

func ValidateName(s string) error {
	if s == "" {
		return errors.New("profile name cannot be empty")
	}
	for _, r := range s {
		if !(r == '-' || r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("invalid profile name %q: use letters, numbers, '.', '_' or '-'", s)
		}
	}
	if s == "." || s == ".." {
		return errors.New("invalid profile name")
	}
	return nil
}

func ValidateProvider(p string) error {
	switch p {
	case "codex", "agy":
		return nil
	default:
		return fmt.Errorf("unsupported provider %q (supported: codex, agy)", p)
	}
}

func Root(provider, name string) (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "profiles", provider, name), nil
}

func Home(provider, name string) (string, error) {
	root, err := Root(provider, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "home"), nil
}

func Exists(provider, name string) bool {
	root, err := Root(provider, name)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(root, "profile.json"))
	return err == nil
}

func Create(provider, name string) (Profile, error) {
	var p Profile
	if err := ValidateProvider(provider); err != nil {
		return p, err
	}
	if err := ValidateName(name); err != nil {
		return p, err
	}
	if Exists(provider, name) {
		return p, fmt.Errorf("profile %s:%s already exists", provider, name)
	}
	root, err := Root(provider, name)
	if err != nil {
		return p, err
	}
	home := filepath.Join(root, "home")
	for _, d := range []string{root, home} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return p, err
		}
		_ = os.Chmod(d, 0700)
	}
	p = Profile{Provider: provider, Name: name, CreatedAt: time.Now()}
	b, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "profile.json"), append(b, '\n'), 0600); err != nil {
		return p, err
	}
	return p, nil
}

func Delete(provider, name string) error {
	root, err := Root(provider, name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	cfg, err := LoadConfig()
	if err == nil && cfg.Defaults[provider] == name {
		delete(cfg.Defaults, provider)
		ps, _ := List()
		for _, p := range ps {
			if p.Provider == provider {
				cfg.Defaults[provider] = p.Name
				break
			}
		}
		_ = SaveConfig(cfg)
	}
	return nil
}

func List() ([]Profile, error) {
	data, err := DataDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(data, "profiles")
	var out []Profile
	for _, provider := range []string{"agy", "codex"} {
		entries, err := os.ReadDir(filepath.Join(base, provider))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(base, provider, e.Name(), "profile.json"))
			if err != nil {
				continue
			}
			var p Profile
			if json.Unmarshal(raw, &p) == nil {
				out = append(out, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider == out[j].Provider {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
		return out[i].Provider < out[j].Provider
	})
	return out, nil
}

func LoadConfig() (Config, error) {
	cfg := Config{Defaults: map[string]string{}}
	dir, err := ConfigDir()
	if err != nil {
		return cfg, err
	}
	path := filepath.Join(dir, "config.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Defaults == nil {
		cfg.Defaults = map[string]string{}
	}
	return cfg, nil
}

func SaveConfig(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(filepath.Join(dir, "config.json"), append(b, '\n'), 0600)
}

func SetDefault(provider, name string) error {
	if !Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.Defaults[provider] = name
	return SaveConfig(cfg)
}

func Default(provider string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Defaults[provider], nil
}

func Get(provider, name string) (Profile, error) {
	var p Profile
	if err := ValidateProvider(provider); err != nil {
		return p, err
	}
	root, err := Root(provider, name)
	if err != nil {
		return p, err
	}
	raw, err := os.ReadFile(filepath.Join(root, "profile.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return p, fmt.Errorf("profile %s:%s does not exist", provider, name)
		}
		return p, err
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	return p, nil
}

type ProfileInfo struct {
	Profile       Profile           `json:"profile"`
	IsDefault     bool              `json:"is_default"`
	RootPath      string            `json:"root_path"`
	HomePath      string            `json:"home_path"`
	ConfigDir     string            `json:"config_dir"`
	DataDir       string            `json:"data_dir"`
	BinaryPath    string            `json:"binary_path"`
	CWD           string            `json:"cwd"`
	UID           int               `json:"uid"`
	GID           int               `json:"gid"`
	IsolationVars map[string]string `json:"isolation_vars"`
	Details       map[string]string `json:"details"`
}

func Inspect(provider, name string) (*ProfileInfo, error) {
	p, err := Get(provider, name)
	if err != nil {
		return nil, err
	}
	root, _ := Root(provider, name)
	home, _ := Home(provider, name)
	d, _ := Default(provider)
	cfgDir, _ := ConfigDir()
	dataDir, _ := DataDir()
	cwd, _ := os.Getwd()
	uid := os.Getuid()
	gid := os.Getgid()

	binPath, _ := exec.LookPath(provider)
	if binPath == "" {
		binPath = "(not found in PATH)"
	}

	info := &ProfileInfo{
		Profile:       p,
		IsDefault:     d == name,
		RootPath:      root,
		HomePath:      home,
		ConfigDir:     cfgDir,
		DataDir:       dataDir,
		BinaryPath:    binPath,
		CWD:           cwd,
		UID:           uid,
		GID:           gid,
		IsolationVars: make(map[string]string),
		Details:       make(map[string]string),
	}

	if provider == "codex" {
		info.IsolationVars["CODEX_HOME"] = home

		cfgPath := filepath.Join(home, "config.toml")
		if raw, err := os.ReadFile(cfgPath); err == nil {
			if strings.Contains(string(raw), "cli_auth_credentials_store") {
				info.Details["config.toml"] = "present (file-backed credentials store)"
			} else {
				info.Details["config.toml"] = "present"
			}
		} else {
			info.Details["config.toml"] = "not found"
		}
		authPath := filepath.Join(home, "auth.json")
		if st, err := os.Stat(authPath); err == nil && st.Size() > 0 {
			info.Details["auth.json"] = fmt.Sprintf("present (%d bytes, credentials stored locally)", st.Size())
		} else {
			info.Details["auth.json"] = "not found (not authenticated yet)"
		}
		if _, err := os.Stat(filepath.Join(home, "sessions")); err == nil {
			info.Details["shared_sessions"] = "linked from host (~/.codex/sessions)"
		}
		if _, err := os.Stat(filepath.Join(home, "history.jsonl")); err == nil {
			info.Details["shared_history"] = "linked from host (~/.codex/history.jsonl)"
		}
	} else if provider == "agy" {
		realHome, _ := os.UserHomeDir()
		info.IsolationVars["HOME"] = home
		info.IsolationVars["XDG_CONFIG_HOME"] = filepath.Join(home, ".config")
		info.IsolationVars["XDG_CACHE_HOME"] = filepath.Join(home, ".cache")
		info.IsolationVars["XDG_DATA_HOME"] = filepath.Join(home, ".local", "share")
		info.IsolationVars["AI_REAL_HOME"] = realHome
		info.IsolationVars["AI_KEYRING_CONTROL_DIR"] = filepath.Join(root, "runtime", "keyring-control")
		if hostDbus := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); hostDbus != "" {
			info.IsolationVars["AI_HOST_DBUS_SESSION_BUS_ADDRESS"] = hostDbus
		} else {
			info.IsolationVars["AI_HOST_DBUS_SESSION_BUS_ADDRESS"] = "(not set on host)"
		}
		info.IsolationVars["GIT_CONFIG_GLOBAL"] = filepath.Join(realHome, ".gitconfig")

		passPath := filepath.Join(root, "keyring.pass")
		if st, err := os.Stat(passPath); err == nil {
			info.Details["keyring.pass"] = fmt.Sprintf("present (mode %04o, isolated local password)", st.Mode().Perm())
		} else {
			info.Details["keyring.pass"] = "not found"
		}

		keyringsDir := filepath.Join(home, ".local", "share", "keyrings")
		if st, err := os.Stat(keyringsDir); err == nil && st.IsDir() {
			loginKeyring := filepath.Join(keyringsDir, "login.keyring")
			if kst, err := os.Stat(loginKeyring); err == nil {
				info.Details["secret_service"] = fmt.Sprintf("initialized (login.keyring present, %d bytes)", kst.Size())
			} else {
				entries, _ := os.ReadDir(keyringsDir)
				info.Details["secret_service"] = fmt.Sprintf("keyrings directory ready (%d entries)", len(entries))
			}
		} else {
			info.Details["secret_service"] = "pending initialization"
		}

		if _, err := os.Stat(filepath.Join(home, ".gitconfig")); err == nil {
			info.Details["git_config"] = "linked from host"
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh")); err == nil {
			info.Details["ssh_config"] = "linked from host"
		}
		if _, err := os.Stat(filepath.Join(home, ".gemini")); err == nil {
			info.Details["shared_gemini"] = "linked from host (~/.gemini conversations & brain)"
		}
	}

	return info, nil
}

func EnsureRandomSecret(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(buf)+"\n"), 0600)
}

