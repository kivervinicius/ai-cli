package codex

import (
	"bufio"
	"context"
	"encoding/base64"
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

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID {
	return "codex"
}

func (a *Adapter) Name() string {
	return "Codex"
}

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              true,
		Conversations:      true,
		Resume:             true,
		CrossAccountResume: true,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "not found in PATH"}
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return err
	}

	// Apply isolation preset policies
	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	if err := security.ApplyIsolation(home, policy); err != nil {
		return err
	}

	// Configure file auth store inside isolated home
	configFile := filepath.Join(home, "config.toml")
	if err := ensureConfigFile(configFile); err != nil {
		return err
	}

	// Link shared non-credential artifacts (sessions, rules, skills, sqlite indices)
	hostHome := security.FindHostHome()
	if hostHome != "" {
		hostCodex := filepath.Join(hostHome, ".codex")
		if _, err := os.Stat(hostCodex); err == nil {
			linkSharedCodexItems(home, hostCodex)
		}
	}

	return nil
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "codex binary not found"}, err
	}
	home, _ := config.ProfileHome(string(a.ID()), p.Name)
	cwd, _ := os.Getwd()
	env := runtime.EnvSet(os.Environ(), map[string]string{"CODEX_HOME": home})
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"login"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	authPath := filepath.Join(home, "auth.json")
	return os.Remove(authPath)
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{
			Status:        "Error",
			Health:        model.HealthUnknown,
			Authenticated: false,
		}
	}

	info := model.AccountInfo{
		Plan:          "ChatGPT Free",
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

	authPath := filepath.Join(home, "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return info
	}

	var authStore struct {
		Tokens struct {
			IDToken     string `json:"id_token"`
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
		AuthMode string `json:"auth_mode"`
	}

	if json.Unmarshal(data, &authStore) != nil || authStore.Tokens.IDToken == "" {
		return info
	}

	info.Authenticated = true
	info.Status = "Authenticated"
	info.Health = model.HealthHealthy
	info.Plan = "ChatGPT Plus"

	// Parse JWT claims safely
	parts := strings.Split(authStore.Tokens.IDToken, ".")
	if len(parts) >= 2 {
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			var claims struct {
				Email      string `json:"email"`
				OpenAIAuth struct {
					PlanType    string `json:"chatgpt_plan_type"`
					ActiveUntil string `json:"chatgpt_subscription_active_until"`
				} `json:"https://api.openai.com/auth"`
			}
			if json.Unmarshal(payload, &claims) == nil {
				if claims.Email != "" {
					info.Email = claims.Email
				}
				if claims.OpenAIAuth.PlanType != "" {
					if claims.OpenAIAuth.PlanType == "pro" {
						info.Plan = "ChatGPT Pro"
					} else {
						info.Plan = "ChatGPT " + strings.Title(strings.ToLower(claims.OpenAIAuth.PlanType))
					}
					if claims.OpenAIAuth.ActiveUntil != "" {
						if t, err := time.Parse(time.RFC3339, claims.OpenAIAuth.ActiveUntil); err == nil {
							info.ExpiresAt = t
						}
					}
				}
			}
		}
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

	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home, _ := config.ProfileHome(string(a.ID()), p.Name)

	candidates := []string{}
	if root != "" {
		candidates = append(candidates, filepath.Join(root, "quota.json"), filepath.Join(root, "usage.json"))
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, "usage.json"), filepath.Join(home, "quota.json"))
	}

	for _, file := range candidates {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var s model.UsageSnapshot
		if json.Unmarshal(data, &s) == nil && s.Status != "" && len(s.Windows) > 0 {
			return s
		}

		var leg struct {
			ModelName string `json:"model_name"`
			FiveHour  struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
			} `json:"five_hour"`
			Weekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
			} `json:"weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetTime != "" || leg.Weekly.ResetTime != "") {
			p5h := leg.FiveHour.PercentLeft
			u5h := 100.0 - p5h
			pWk := leg.Weekly.PercentLeft
			uWk := 100.0 - pWk

			return model.UsageSnapshot{
				ProviderID: string(a.ID()),
				ProfileID:  p.Name,
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  time.Now(),
				Windows: []model.UsageWindow{
					{
						Kind:             "5h",
						RemainingPercent: &p5h,
						UsedPercent:      &u5h,
						ResetDescription: leg.FiveHour.ResetTime,
					},
					{
						Kind:             "weekly",
						RemainingPercent: &pWk,
						UsedPercent:      &uWk,
						ResetDescription: leg.Weekly.ResetTime,
					},
				},
			}
		}
	}

	return snap
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session

	homeCandidates := []string{}
	if h, err := config.ProfileHome(string(a.ID()), p.Name); err == nil {
		homeCandidates = append(homeCandidates, h, filepath.Join(h, ".codex"))
	}
	if root, err := config.ProfileRoot(string(a.ID()), p.Name); err == nil {
		homeCandidates = append(homeCandidates, root)
	}
	if hostHome := security.FindHostHome(); hostHome != "" {
		homeCandidates = append(homeCandidates, filepath.Join(hostHome, ".codex"), hostHome)
	}
	if userProf := os.Getenv("USERPROFILE"); userProf != "" {
		homeCandidates = append(homeCandidates, filepath.Join(userProf, ".codex"), userProf)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		homeCandidates = append(homeCandidates, filepath.Join(appData, "codex"), filepath.Join(appData, "OpenAI", "Codex"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		homeCandidates = append(homeCandidates, filepath.Join(localAppData, "codex"), filepath.Join(localAppData, "OpenAI", "Codex"))
	}

	seen := make(map[string]bool)
	for _, h := range homeCandidates {
		indexFile := filepath.Join(h, "session_index.jsonl")
		f, err := os.Open(indexFile)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry struct {
				ID         string `json:"id"`
				ThreadName string `json:"thread_name"`
				UpdatedAt  string `json:"updated_at"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil && entry.ID != "" {
				if seen[entry.ID] {
					continue
				}
				seen[entry.ID] = true

				t, _ := time.Parse(time.RFC3339, entry.UpdatedAt)
				title := entry.ThreadName
				if title == "" {
					title = "Codex Thread " + entry.ID[:8]
				}
				sessions = append(sessions, model.Session{
					ProviderID:      string(a.ID()),
					ProfileID:       p.Name,
					ID:              entry.ID,
					Title:           title,
					Workspace:       workspace,
					CreatedAt:       t,
					UpdatedAt:       t,
					ResumeSupported: true,
				})
			}
		}
		f.Close()
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	resumeArgs := []string{"resume", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}

func ensureConfigFile(path string) error {
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

	content := string(raw)
	if !strings.Contains(content, "cli_auth_credentials_store") {
		content += "\ncli_auth_credentials_store = \"file\"\n"
		return os.WriteFile(path, []byte(content), 0600)
	}
	return nil
}

func linkSharedCodexItems(profileHome, hostCodex string) {
	items := []string{
		"session_index.jsonl",
		"thread_history_1.sqlite",
		"thread_history_1.sqlite-wal",
		"thread_history_1.sqlite-shm",
		"rules",
		"skills",
		"customizations",
	}

	for _, item := range items {
		src := filepath.Join(hostCodex, item)
		dst := filepath.Join(profileHome, item)

		if _, err := os.Stat(src); err != nil {
			continue
		}

		if fi, err := os.Lstat(dst); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dst)
				if err == nil && target == src {
					continue
				}
				_ = os.Remove(dst)
			} else {
				continue
			}
		}

		_ = os.Symlink(src, dst)
	}
}
