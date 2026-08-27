package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

// Adapter implements the Provider interface for Anthropic Claude Code.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID { return model.ProviderClaude }
func (a *Adapter) Name() string         { return "Claude Code" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              true,
		Conversations:      true,
		Resume:             true,
		CrossAccountResume: false,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := runtime.LookPath("claude")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "binary not found in PATH"}
	}
	out, err := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	version := strings.TrimSpace(out)
	if err != nil && version == "" {
		return model.DetectionResult{Installed: true, BinaryPath: bin, Version: "unknown"}
	}
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    version,
	}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	claudeDir := filepath.Join(home, ".claude")
	configDir := filepath.Join(home, ".config", "claude")
	for _, d := range []string{home, claudeDir, configDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}

	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	return security.ApplyIsolation(home, policy)
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := runtime.LookPath("claude")
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "claude binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	envMap := map[string]string{
		"HOME":              home,
		"CLAUDE_CONFIG_DIR": filepath.Join(home, ".claude"),
		"XDG_CONFIG_HOME":   filepath.Join(home, ".config"),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}

	env := runtime.EnvSet(os.Environ(), envMap)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"login"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.Remove(filepath.Join(home, ".claude.json"))
	_ = os.RemoveAll(filepath.Join(home, ".claude"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "Claude Pro / Team",
		Status:        "Not authenticated",
		Health:        model.HealthAuthRequired,
		Authenticated: false,
		Usage: model.UsageSnapshot{
			ProviderID: string(a.ID()),
			ProfileID:  p.Name,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		},
	}

	// 1. Check ~/.claude.json
	claudeJson := filepath.Join(home, ".claude.json")
	if data, err := os.ReadFile(claudeJson); err == nil {
		var cData struct {
			OAuthAccount struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"oauthAccount"`
			UserID string `json:"userID"`
		}
		if json.Unmarshal(data, &cData) == nil {
			if cData.OAuthAccount.Email != "" {
				info.Email = cData.OAuthAccount.Email
				info.Authenticated = true
				info.Status = "Authenticated"
				info.Health = model.HealthHealthy
				return info
			}
			if cData.UserID != "" {
				info.Email = cData.UserID
				info.Authenticated = true
				info.Status = "Authenticated"
				info.Health = model.HealthHealthy
				return info
			}
		}
	}

	// 2. Check credentials.json in .claude/
	credFile := filepath.Join(home, ".claude", "credentials.json")
	if st, err := os.Stat(credFile); err == nil && st.Size() > 0 {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
	}

	return info
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	snap := model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}

	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return snap
	}

	quotaFile := filepath.Join(root, "home", "usage.json")
	if data, err := os.ReadFile(quotaFile); err == nil {
		var s model.UsageSnapshot
		if json.Unmarshal(data, &s) == nil {
			return s
		}
	}

	return snap
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(root, "home", ".claude", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return sessions, nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sessions = append(sessions, model.Session{
			ProviderID:      string(a.ID()),
			ProfileID:       p.Name,
			ID:              e.Name(),
			Title:           "Claude Session " + e.Name(),
			Workspace:       workspace,
			UpdatedAt:       time.Now(),
			ResumeSupported: true,
		})
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	// Claude official resume syntax: claude --resume <SESSION_ID>
	resumeArgs := []string{"--resume", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}
