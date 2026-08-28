package agy

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID {
	return "agy"
}

func (a *Adapter) Name() string {
	return "AGY"
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
	bin, err := runtime.LookPath("agy")
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
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	keyrings := filepath.Join(home, ".local", "share", "keyrings")
	geminiDir := filepath.Join(home, ".gemini")

	for _, d := range []string{root, home, keyrings, geminiDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}

	// Apply isolation preset policies
	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	if err := security.ApplyIsolation(home, policy); err != nil {
		return err
	}

	// Link shared non-credential artifacts
	hostHome := security.FindHostHome()
	if hostHome != "" {
		hostGemini := filepath.Join(hostHome, ".gemini")
		if _, err := os.Stat(hostGemini); err == nil {
			linkSharedAgyItems(home, hostGemini)
		}
	}

	return nil
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := runtime.LookPath("agy")
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "agy binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	internalBin, err := runtime.InternalBinDir()
	if err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}

	// Ensure keyring password file
	pwFile := filepath.Join(root, "keyring.pass")
	if _, err := os.Stat(pwFile); err != nil {
		_ = os.WriteFile(pwFile, []byte("agy-keyring-secret\n"), 0600)
	}

	// Set isolated environment
	envOverrides := map[string]string{
		"HOME":             home,
		"XDG_CONFIG_HOME":  filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":   filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":    filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":   filepath.Join(home, ".local", "state"),
		"PATH":             internalBin + ":" + os.Getenv("PATH"),
		"BROWSER":          filepath.Join(internalBin, "ai-browser"),
	}

	env := runtime.EnvSet(os.Environ(), envOverrides, "DBUS_SESSION_BUS_ADDRESS")

	// Wrap execution in isolated D-Bus daemon if dbus-run-session is available
	var runBin string
	var runArgs []string

	if dbusPath, err := runtime.LookPath("dbus-run-session"); err == nil {
		runBin = dbusPath
		runArgs = []string{"--", bin}
		runArgs = append(runArgs, args...)
	} else {
		runBin = bin
		runArgs = args
	}

	return runtime.RunInteractive(runBin, runArgs, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, nil)
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.RemoveAll(filepath.Join(home, ".gemini", "google_accounts.json"))
	_ = os.RemoveAll(filepath.Join(home, ".gemini", "antigravity-oauth-token"))
	_ = os.RemoveAll(filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt"))
	_ = os.RemoveAll(filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"))
	_ = os.RemoveAll(filepath.Join(home, ".local", "share", "keyrings"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "Google AI Pro",
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

	// Read email and plan from quota.json or usage.json if available
	quotaCandidates := []string{
		filepath.Join(root, "quota.json"),
		filepath.Join(home, "quota.json"),
		filepath.Join(root, "usage.json"),
		filepath.Join(home, "usage.json"),
	}
	for _, qFile := range quotaCandidates {
		if data, err := os.ReadFile(qFile); err == nil {
			var q struct {
				Account string `json:"account"`
				Email   string `json:"email"`
				Plan    string `json:"plan"`
			}
			if json.Unmarshal(data, &q) == nil {
				if q.Account != "" {
					info.Email = q.Account
				} else if q.Email != "" {
					info.Email = q.Email
				}
				if q.Plan != "" {
					info.Plan = q.Plan
				}
				break
			}
		}
	}

	// 1. Check antigravity-oauth-token in antigravity-cli or .gemini
	tokenFiles := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"),
		filepath.Join(home, ".gemini", "antigravity-oauth-token"),
	}
	for _, tf := range tokenFiles {
		if data, err := os.ReadFile(tf); err == nil && len(data) > 0 {
			var tok struct {
				Token struct {
					AccessToken  string `json:"access_token"`
					RefreshToken string `json:"refresh_token"`
					Expiry       string `json:"expiry"`
				} `json:"token"`
			}
			if json.Unmarshal(data, &tok) == nil && (tok.Token.AccessToken != "" || tok.Token.RefreshToken != "") {
				info.Authenticated = true
				info.Status = "Authenticated"
				info.Health = model.HealthHealthy
				if info.Email == "" {
					info.Email = p.Name
				}
				return info
			}
		}
	}

	// 2. Check google_accounts.json in profile home
	accountsFile := filepath.Join(home, ".gemini", "google_accounts.json")
	if data, err := os.ReadFile(accountsFile); err == nil {
		var acc struct {
			Active string `json:"active"`
		}
		if json.Unmarshal(data, &acc) == nil && acc.Active != "" {
			info.Email = acc.Active
			info.Authenticated = true
			info.Status = "Authenticated"
			info.Health = model.HealthHealthy
			return info
		}
	}

	// 3. Check jetski_state.pbtxt
	jetskiFile := filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
	if data, err := os.ReadFile(jetskiFile); err == nil && len(data) > 0 {
		matches := emailRegex.FindAllString(string(data), -1)
		if len(matches) > 0 {
			info.Email = matches[0]
			info.Authenticated = true
			info.Status = "Authenticated"
			info.Health = model.HealthHealthy
			return info
		}
	}

	// 4. Check keyring files (if login.keyring exists, credentials are stored)
	keyringsDir := filepath.Join(home, ".local", "share", "keyrings")
	if _, err := os.Stat(filepath.Join(keyringsDir, "login.keyring")); err == nil {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		if info.Email == "" {
			info.Email = p.Name
		}
		return info
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
				ResetsIn    string  `json:"resets_in"`
			} `json:"five_hour"`
			Weekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"weekly"`
			ClaudeFiveHour struct {
				PercentLeft float64 `json:"percent_left"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"claude_five_hour"`
			ClaudeWeekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"claude_weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetsIn != "" || leg.ClaudeFiveHour.PercentLeft > 0 || leg.ClaudeWeekly.PercentLeft > 0) {
			p5h := leg.FiveHour.PercentLeft
			u5h := 100.0 - p5h
			pWk := leg.Weekly.PercentLeft
			uWk := 100.0 - pWk

			windows := []model.UsageWindow{
				{
					Kind:             "5h",
					RemainingPercent: &p5h,
					UsedPercent:      &u5h,
					ResetDescription: leg.FiveHour.ResetsIn,
				},
				{
					Kind:             "weekly",
					RemainingPercent: &pWk,
					UsedPercent:      &uWk,
					ResetDescription: leg.Weekly.ResetsIn,
				},
			}

			if leg.ClaudeFiveHour.PercentLeft > 0 || leg.ClaudeFiveHour.ResetsIn != "" {
				pC5h := leg.ClaudeFiveHour.PercentLeft
				uC5h := 100.0 - pC5h
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_5h",
					RemainingPercent: &pC5h,
					UsedPercent:      &uC5h,
					ResetDescription: leg.ClaudeFiveHour.ResetsIn,
				})
			}

			if leg.ClaudeWeekly.PercentLeft > 0 || leg.ClaudeWeekly.ResetsIn != "" {
				pCWk := leg.ClaudeWeekly.PercentLeft
				uCWk := 100.0 - pCWk
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_weekly",
					RemainingPercent: &pCWk,
					UsedPercent:      &uCWk,
					ResetDescription: leg.ClaudeWeekly.ResetsIn,
				})
			}

			return model.UsageSnapshot{
				ProviderID: string(a.ID()),
				ProfileID:  p.Name,
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  time.Now(),
				Windows:    windows,
			}
		}
	}

	return snap
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session

	homeCandidates := []string{}
	if root, err := config.ProfileRoot(string(a.ID()), p.Name); err == nil {
		homeCandidates = append(homeCandidates, filepath.Join(root, "home"), root)
	}
	if hostHome := security.FindHostHome(); hostHome != "" {
		homeCandidates = append(homeCandidates, hostHome)
	}
	if userProf := os.Getenv("USERPROFILE"); userProf != "" {
		homeCandidates = append(homeCandidates, userProf)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		homeCandidates = append(homeCandidates, appData, filepath.Join(appData, "antigravity-cli"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		homeCandidates = append(homeCandidates, localAppData, filepath.Join(localAppData, "antigravity-cli"))
	}

	seen := make(map[string]bool)
	for _, h := range homeCandidates {
		histCandidates := []string{
			filepath.Join(h, ".gemini", "antigravity-cli", "history.jsonl"),
			filepath.Join(h, "antigravity-cli", "history.jsonl"),
			filepath.Join(h, "history.jsonl"),
		}
		for _, histFile := range histCandidates {
			f, err := os.Open(histFile)
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
				Display        string `json:"display"`
				Timestamp      int64  `json:"timestamp"`
				Workspace      string `json:"workspace"`
				ConversationID string `json:"conversationId"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil && entry.ConversationID != "" {
				if seen[entry.ConversationID] {
					continue
				}
				seen[entry.ConversationID] = true

				t := time.UnixMilli(entry.Timestamp)
				title := entry.Display
				if strings.HasPrefix(title, "/") {
					title = "AGY Session " + entry.ConversationID[:8]
				}
				ws := entry.Workspace
				if ws == "" {
					ws = workspace
				}
				sessions = append(sessions, model.Session{
					ProviderID:      string(a.ID()),
					ProfileID:       p.Name,
					ID:              entry.ConversationID,
					Title:           title,
					Workspace:       ws,
					CreatedAt:       t,
					UpdatedAt:       t,
					ResumeSupported: true,
				})
			}
		}
		f.Close()
		}
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	resumeArgs := []string{"--conversation=" + sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}

func linkSharedAgyItems(profileHome, hostGemini string) {
	items := []string{
		"antigravity-cli/history.jsonl",
		"antigravity-cli/conversation_summaries.db",
		"antigravity-cli/settings.json",
		"antigravity-cli/installed_extensions.json",
		"antigravity-cli/customizations",
	}

	for _, item := range items {
		src := filepath.Join(hostGemini, item)
		dst := filepath.Join(profileHome, ".gemini", item)

		if _, err := os.Stat(src); err != nil {
			continue
		}

		_ = security.SafeLinkOrCopy(src, dst)
	}
}
