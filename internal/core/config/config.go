package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

const CurrentConfigVersion = 1

var (
	configMu sync.RWMutex
)

// Config represents persistent application settings and user preferences.
type Config struct {
	ConfigVersion     int                                 `json:"config_version"`
	Defaults          map[string]string                   `json:"defaults"`            // provider -> profile
	Priorities        map[string]map[string]int           `json:"priorities,omitempty"` // provider -> profile -> priority
	Disabled          map[string]map[string]bool          `json:"disabled,omitempty"`   // provider -> profile -> true
	Labels            map[string]map[string][]string      `json:"labels,omitempty"`     // provider -> profile -> tags
	Strategy          string                              `json:"strategy"`             // best-capacity, least-used, round-robin, sticky
	StickyTTL         string                              `json:"sticky_ttl,omitempty"` // e.g. "30m"
	IsolationPreset   model.IsolationPreset               `json:"isolation_preset"`    // developer, strict, compat
	Bindings          map[string]map[string]string        `json:"bindings,omitempty"`   // workspace -> provider -> profile
	AutomaticFallback bool                                `json:"automatic_fallback"`
	MaxConcurrency    int                                 `json:"max_concurrency"`
}

// NewDefaultConfig returns a well-configured default Configuration.
func NewDefaultConfig() Config {
	return Config{
		ConfigVersion:     CurrentConfigVersion,
		Defaults:          make(map[string]string),
		Priorities:        make(map[string]map[string]int),
		Disabled:          make(map[string]map[string]bool),
		Labels:            make(map[string]map[string][]string),
		Strategy:          "best-capacity",
		StickyTTL:         "30m",
		IsolationPreset:   model.IsolationDeveloper,
		Bindings:          make(map[string]map[string]string),
		AutomaticFallback: true,
		MaxConcurrency:    4,
	}
}

func getBaseHome() string {
	if v := os.Getenv("AI_REAL_HOME"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		if idx := strings.Index(h, "/.local/share/ai-manager/profiles"); idx != -1 {
			return h[:idx]
		}
		if idx := strings.Index(h, "/.local/share/ai-cli/profiles"); idx != -1 {
			return h[:idx]
		}
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

// ConfigDir returns the XDG-compliant config directory with ai-manager fallback.
func ConfigDir() (string, error) {
	if v := os.Getenv("AI_MANAGER_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	configBase := filepath.Join(home, ".config")
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && !strings.Contains(xdg, "/profiles/") {
		configBase = xdg
	}

	pCli := filepath.Join(configBase, "ai-cli")
	pMgr := filepath.Join(configBase, "ai-manager")

	// If ai-cli exists, use it
	if _, err := os.Stat(filepath.Join(pCli, "config.json")); err == nil {
		return pCli, nil
	}
	// Fallback to existing ai-manager config
	if _, err := os.Stat(filepath.Join(pMgr, "config.json")); err == nil {
		return pMgr, nil
	}

	return pCli, nil
}

// DataDir returns the XDG-compliant data directory with ai-manager fallback.
func DataDir() (string, error) {
	if v := os.Getenv("AI_MANAGER_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	dataBase := filepath.Join(home, ".local", "share")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && !strings.Contains(xdg, "/profiles/") {
		dataBase = xdg
	}

	pCli := filepath.Join(dataBase, "ai-cli")
	pMgr := filepath.Join(dataBase, "ai-manager")

	// If ai-cli profiles directory exists, use it
	if _, err := os.Stat(filepath.Join(pCli, "profiles")); err == nil {
		return pCli, nil
	}
	// Fallback to existing ai-manager directory
	if _, err := os.Stat(filepath.Join(pMgr, "profiles")); err == nil {
		return pMgr, nil
	}

	return pCli, nil
}

// StateDir returns the XDG-compliant state directory.
func StateDir() (string, error) {
	if v := os.Getenv("AI_MANAGER_STATE_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_STATE_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	stateBase := filepath.Join(home, ".local", "state")
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" && !strings.Contains(xdg, "/profiles/") {
		stateBase = xdg
	}

	pCli := filepath.Join(stateBase, "ai-cli")
	pMgr := filepath.Join(stateBase, "ai-manager")

	if _, err := os.Stat(pCli); err == nil {
		return pCli, nil
	}
	if _, err := os.Stat(pMgr); err == nil {
		return pMgr, nil
	}

	return pCli, nil
}

// ProfileRoot returns the profile root directory.
func ProfileRoot(provider, name string) (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "profiles", provider, name), nil
}

// ProfileHome returns the isolated home directory for a profile.
func ProfileHome(provider, name string) (string, error) {
	root, err := ProfileRoot(provider, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "home"), nil
}

// LoadConfig reads the configuration file with migration support.
func LoadConfig() (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	cfg := NewDefaultConfig()
	dir, err := ConfigDir()
	if err != nil {
		return cfg, err
	}

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	// Legacy config migration support: handle old defaults format if present
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err == nil {
		if rawMap["defaults"] != nil {
			if defs, ok := rawMap["defaults"].(map[string]interface{}); ok {
				for k, v := range defs {
					if strVal, ok := v.(string); ok {
						cfg.Defaults[k] = strVal
					}
				}
			}
		}
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Defaults == nil {
		cfg.Defaults = make(map[string]string)
	}
	if cfg.Priorities == nil {
		cfg.Priorities = make(map[string]map[string]int)
	}
	if cfg.Disabled == nil {
		cfg.Disabled = make(map[string]map[string]bool)
	}
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]map[string][]string)
	}
	if cfg.Bindings == nil {
		cfg.Bindings = make(map[string]map[string]string)
	}
	if cfg.IsolationPreset == "" {
		cfg.IsolationPreset = model.IsolationDeveloper
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "best-capacity"
	}
	if cfg.StickyTTL == "" {
		cfg.StickyTTL = "30m"
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4
	}

	return cfg, nil
}

// SaveConfig atomically persists configuration to disk.
func SaveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	tmpPath := fmt.Sprintf("%s.tmp.%d", configPath, time.Now().UnixNano())

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, append(data, '\n'), 0600); err != nil {
		return err
	}

	return os.Rename(tmpPath, configPath)
}

// GetDefaultProfile returns the default profile for a provider.
func GetDefaultProfile(provider string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Defaults[provider], nil
}

// SetDefaultProfile sets the default profile for a provider.
func SetDefaultProfile(provider, name string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.Defaults[provider] = name
	return SaveConfig(cfg)
}

// BindWorkspace associates a workspace with a provider profile.
func BindWorkspace(workspace, provider, profileName string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if cfg.Bindings[absWs] == nil {
		cfg.Bindings[absWs] = make(map[string]string)
	}
	cfg.Bindings[absWs][provider] = profileName
	return SaveConfig(cfg)
}

// UnbindWorkspace removes workspace binding for a provider.
func UnbindWorkspace(workspace, provider string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if cfg.Bindings[absWs] != nil {
		delete(cfg.Bindings[absWs], provider)
		if len(cfg.Bindings[absWs]) == 0 {
			delete(cfg.Bindings, absWs)
		}
		return SaveConfig(cfg)
	}
	return nil
}

// GetBinding returns the bound profile for a given workspace and provider.
func GetBinding(workspace, provider string) string {
	cfg, err := LoadConfig()
	if err != nil {
		return ""
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if b, ok := cfg.Bindings[absWs]; ok {
		return b[provider]
	}
	return ""
}

// ValidateProfileName validates that a profile name uses allowed characters.
func ValidateProfileName(name string) error {
	if name == "" {
		return errors.New("profile name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("profile name too long (max 64 characters)")
	}
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return fmt.Errorf("invalid character %q in profile name (allowed: a-z, A-Z, 0-9, -, _, .)", string(ch))
	}
	return nil
}

// Validate performs structural validation of configuration settings.
func (c Config) Validate() []string {
	var issues []string
	if c.Strategy != "best-capacity" && c.Strategy != "least-used" && c.Strategy != "round-robin" && c.Strategy != "sticky" {
		issues = append(issues, fmt.Sprintf("unknown strategy %q (allowed: best-capacity, least-used, round-robin, sticky)", c.Strategy))
	}
	if c.IsolationPreset != model.IsolationStrict && c.IsolationPreset != model.IsolationDeveloper && c.IsolationPreset != model.IsolationCompat {
		issues = append(issues, fmt.Sprintf("unknown isolation preset %q (allowed: developer, strict, compat)", c.IsolationPreset))
	}
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 32 {
		issues = append(issues, "max_concurrency must be between 1 and 32")
	}
	return issues
}
