package agy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
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

	// Set isolated environment
	envOverrides := map[string]string{
		"HOME":                             home,
		"XDG_CONFIG_HOME":                  filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":                   filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":                    filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":                   filepath.Join(home, ".local", "state"),
		"PATH":                             runtime.EnhancedPATH(internalBin, filepath.Dir(bin)),
		"BROWSER":                          filepath.Join(internalBin, "ai-browser"),
		"AI_HOST_DBUS_SESSION_BUS_ADDRESS": os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
		"PYTHON_KEYRING_BACKEND":           "keyring.backends.null.Keyring",
	}

	env := runtime.EnvSet(os.Environ(), envOverrides, "DBUS_SESSION_BUS_ADDRESS", "GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID")

	wrappedBin, wrappedArgs := runtime.WrapWithIsolatedSecretService(bin, args)
	return runtime.RunInteractive(wrappedBin, wrappedArgs, env, cwd)
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
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("AGY GetUsage: checking cached files", "profile", p.Name)
	}
	if fileSnap, ok := a.readCachedQuotaFiles(p); ok {
		if quota.NewEngine(quota.DefaultTTL).Trustworthy(fileSnap) {
			if debug {
				slog.Debug("AGY GetUsage: returning trustworthy cached snapshot", "profile", p.Name, "status", fileSnap.Status, "fetchedAt", fileSnap.FetchedAt)
			}
			return fileSnap
		}
		if debug {
			slog.Debug("AGY GetUsage: cached snapshot not trustworthy", "profile", p.Name, "fetchedAt", fileSnap.FetchedAt, "age", time.Since(fileSnap.FetchedAt))
		}
	}
	if debug {
		slog.Debug("AGY GetUsage: trying live fetch", "profile", p.Name)
	}
	if live, ok := a.fetchLiveQuota(ctx, p); ok {
		_ = quota.NewEngine(quota.DefaultTTL).SaveUsage(live)
		if debug {
			slog.Debug("AGY GetUsage: returning live snapshot", "profile", p.Name, "status", live.Status, "windows", len(live.Windows))
		}
		return live
	}
	if debug {
		slog.Debug("AGY GetUsage: returning unknown (no cached or live data)", "profile", p.Name)
	}
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("AGY RefreshUsage: trying live fetch first", "profile", p.Name)
	}
	if live, ok := a.fetchLiveQuota(ctx, p); ok {
		_ = quota.NewEngine(quota.DefaultTTL).SaveUsage(live)
		if debug {
			slog.Debug("AGY RefreshUsage: returning live snapshot", "profile", p.Name, "status", live.Status, "windows", len(live.Windows))
		}
		return live
	}
	if debug {
		slog.Debug("AGY RefreshUsage: live fetch failed, trying cached", "profile", p.Name)
	}
	if fileSnap, ok := a.readCachedQuotaFiles(p); ok {
		if debug {
			slog.Debug("AGY RefreshUsage: returning cached snapshot", "profile", p.Name, "status", fileSnap.Status, "fetchedAt", fileSnap.FetchedAt)
		}
		return fileSnap
	}
	if debug {
		slog.Debug("AGY RefreshUsage: returning unknown (no cached or live data)", "profile", p.Name)
	}
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func (a *Adapter) readCachedQuotaFiles(p model.Profile) (model.UsageSnapshot, bool) {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home, _ := config.ProfileHome(string(a.ID()), p.Name)

	candidates := []string{}
	if root != "" {
		candidates = append(candidates, filepath.Join(root, "usage.json"), filepath.Join(root, "quota.json"))
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, "usage.json"), filepath.Join(home, "quota.json"))
	}

	if debug {
		slog.Debug("AGY readCachedQuotaFiles: checking candidates", "profile", p.Name, "candidates", candidates)
	}

	for _, file := range candidates {
		data, err := os.ReadFile(file)
		if err != nil {
			if debug {
				slog.Debug("AGY readCachedQuotaFiles: file read error", "profile", p.Name, "file", file, "err", err)
			}
			continue
		}

		var s model.UsageSnapshot
		if json.Unmarshal(data, &s) == nil && s.Status != "" && len(s.Windows) > 0 {
			// Mark status as CACHED if it was originally LIVE (consistent with GetCachedUsage)
			if s.Status == model.UsageLive {
				s.Status = model.UsageCached
			}
			if debug {
				slog.Debug("AGY readCachedQuotaFiles: found UsageSnapshot", "profile", p.Name, "file", file, "status", s.Status, "fetchedAt", s.FetchedAt, "windows", len(s.Windows), "account", s.Account)
			}
			return s, true
		}

		var leg struct {
			Account   string `json:"account"`
			Email     string `json:"email"`
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
				PercentLeft *float64 `json:"percent_left"`
				ResetsIn    string   `json:"resets_in"`
			} `json:"claude_five_hour"`
			ClaudeWeekly struct {
				PercentLeft *float64 `json:"percent_left"`
				ResetsIn    string   `json:"resets_in"`
			} `json:"claude_weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetsIn != "" || leg.ClaudeFiveHour.PercentLeft != nil || leg.ClaudeWeekly.PercentLeft != nil || leg.ClaudeFiveHour.ResetsIn != "" || leg.ClaudeWeekly.ResetsIn != "") {
			if debug {
				slog.Debug("AGY readCachedQuotaFiles: found legacy format", "profile", p.Name, "file", file, "account", firstNonEmpty(leg.Account, leg.Email), "gemini5h", leg.FiveHour.PercentLeft, "geminiWeekly", leg.Weekly.PercentLeft)
			}
			p5h := agyClampPercent(leg.FiveHour.PercentLeft)
			u5h := 100 - p5h
			pWk := agyClampPercent(leg.Weekly.PercentLeft)
			uWk := 100 - pWk

			windows := []model.UsageWindow{
				{
					Kind:             "5h",
					Group:            "gemini",
					RemainingPercent: &p5h,
					UsedPercent:      &u5h,
					ResetDescription: leg.FiveHour.ResetsIn,
				},
				{
					Kind:             "weekly",
					Group:            "gemini",
					RemainingPercent: &pWk,
					UsedPercent:      &uWk,
					ResetDescription: leg.Weekly.ResetsIn,
				},
			}

			// Exhausted (0%) is not absent: emit Claude when the legacy section exists.
			hasClaude := leg.ClaudeFiveHour.PercentLeft != nil || leg.ClaudeWeekly.PercentLeft != nil ||
				leg.ClaudeFiveHour.ResetsIn != "" || leg.ClaudeWeekly.ResetsIn != ""
			if hasClaude {
				c5hVal := 0.0
				if leg.ClaudeFiveHour.PercentLeft != nil {
					c5hVal = *leg.ClaudeFiveHour.PercentLeft
				}
				pC5h := agyClampPercent(c5hVal)
				uC5h := 100 - pC5h
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_5h",
					Group:            "claude_gpt",
					RemainingPercent: &pC5h,
					UsedPercent:      &uC5h,
					ResetDescription: leg.ClaudeFiveHour.ResetsIn,
				})
				cWkVal := 0.0
				if leg.ClaudeWeekly.PercentLeft != nil {
					cWkVal = *leg.ClaudeWeekly.PercentLeft
				}
				pCWk := agyClampPercent(cWkVal)
				uCWk := 100 - pCWk
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_weekly",
					Group:            "claude_gpt",
					RemainingPercent: &pCWk,
					UsedPercent:      &uCWk,
					ResetDescription: leg.ClaudeWeekly.ResetsIn,
				})
			}

			snap := model.UsageSnapshot{
				ProviderID: string(a.ID()),
				ProfileID:  p.Name,
				Account:    firstNonEmpty(leg.Account, leg.Email),
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  fileModTime(file),
				Windows:    windows,
			}
			if debug {
				slog.Debug("AGY readCachedQuotaFiles: returning legacy snapshot", "profile", p.Name, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
			}
			return snap, true
		}
		if debug {
			slog.Debug("AGY readCachedQuotaFiles: file not recognized", "profile", p.Name, "file", file)
		}
	}

	if debug {
		slog.Debug("AGY readCachedQuotaFiles: no valid cache found", "profile", p.Name)
	}
	return model.UsageSnapshot{}, false
}

func fileModTime(path string) time.Time {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func agyClampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func (a *Adapter) fetchLiveQuota(ctx context.Context, p model.Profile) (model.UsageSnapshot, bool) {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("AGY fetchLiveQuota: starting", "profile", p.Name)
	}

	if !a.InspectAuth(ctx, p).Authenticated {
		if debug {
			slog.Debug("AGY fetchLiveQuota: not authenticated", "profile", p.Name)
		}
		return model.UsageSnapshot{}, false
	}
	if err := a.Prepare(ctx, p); err != nil {
		if debug {
			slog.Debug("AGY fetchLiveQuota: prepare failed", "profile", p.Name, "err", err)
		}
		return model.UsageSnapshot{}, false
	}
	bin, err := runtime.LookPath("agy")
	if err != nil {
		if debug {
			slog.Debug("AGY fetchLiveQuota: agy binary not found", "profile", p.Name)
		}
		return model.UsageSnapshot{}, false
	}
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		if debug {
			slog.Debug("AGY fetchLiveQuota: profile root failed", "profile", p.Name, "err", err)
		}
		return model.UsageSnapshot{}, false
	}
	home := filepath.Join(root, "home")
	internalBin, err := runtime.InternalBinDir()
	if err != nil {
		if debug {
			slog.Debug("AGY fetchLiveQuota: internal bin dir failed", "profile", p.Name, "err", err)
		}
		return model.UsageSnapshot{}, false
	}
	envOverrides := map[string]string{
		"HOME":                             home,
		"XDG_CONFIG_HOME":                  filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":                   filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":                    filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":                   filepath.Join(home, ".local", "state"),
		"PATH":                             runtime.EnhancedPATH(internalBin, filepath.Dir(bin)),
		"BROWSER":                          filepath.Join(internalBin, "ai-browser"),
		"AI_HOST_DBUS_SESSION_BUS_ADDRESS": os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
		"PYTHON_KEYRING_BACKEND":           "keyring.backends.null.Keyring",
	}
	env := runtime.EnvSet(os.Environ(), envOverrides, "DBUS_SESSION_BUS_ADDRESS", "GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID")
	args := []string{"--output-format", "text", "--print-timeout", "12s", "--print=/quota"}
	// Quota probes are deliberately non-interactive and must never initialize
	// Secret Service. AGY's interactive/login path still uses the isolated
	// keyring in Run above, but a background quota refresh must not prompt for
	// a password or create a new keyring daemon. With the session bus and
	// keyring variables removed from env, AGY can use its profile token/file
	// storage; if that is unavailable the probe fails closed and the caller
	// exposes UNKNOWN/last-known data instead of blocking the user.

	if debug {
		slog.Debug("AGY fetchLiveQuota: running agy CLI", "profile", p.Name, "home", home, "bin", bin)
	}

	fetchCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		fetchCtx, cancel = context.WithTimeout(ctx, 12*time.Second)
		defer cancel()
	}
	out, err := runtime.RunCommandCapture(fetchCtx, bin, args, env, home)
	if err != nil {
		if debug {
			slog.Debug("AGY fetchLiveQuota: agy CLI failed", "profile", p.Name, "err", err, "output", out)
		}
		return model.UsageSnapshot{}, false
	}
	if debug {
		slog.Debug("AGY fetchLiveQuota: agy CLI output", "profile", p.Name, "output", out)
	}
	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		if debug {
			slog.Debug("AGY fetchLiveQuota: parseAgyQuotaOutput failed", "profile", p.Name, "output", out)
		}
		return model.UsageSnapshot{}, false
	}
	info := a.InspectAuth(ctx, p)
	if debug {
		slog.Debug("AGY fetchLiveQuota: success", "profile", p.Name, "email", info.Email, "windows", len(windows))
	}
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Account:    firstNonEmpty(info.Email, p.Name),
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		ModelName:  "Gemini Flash, Gemini Pro / Claude, GPT",
		FetchedAt:  time.Now(),
		Windows:    windows,
	}, true
}

func parseAgyQuotaOutput(output string) ([]model.UsageWindow, bool) {
	var windows []model.UsageWindow
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		groupName := strings.ToLower(strings.TrimSpace(fields[0]))
		label := strings.ToLower(strings.TrimSpace(fields[1]))
		pct, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(fields[2]), "%"), 64)
		if err != nil {
			continue
		}
		pct = agyClampPercent(pct)
		used := 100 - pct
		reset := ""
		if len(fields) >= 4 {
			reset = formatAgyReset(strings.TrimSpace(fields[3]))
		}
		group := "gemini"
		if strings.Contains(groupName, "claude") || strings.Contains(groupName, "gpt") {
			group = "claude_gpt"
		}
		kind := "weekly"
		if strings.Contains(label, "five hour") || strings.Contains(label, "5h") {
			kind = "5h"
			if group == "claude_gpt" {
				kind = "claude_5h"
			}
		} else if strings.Contains(label, "weekly") {
			if group == "claude_gpt" {
				kind = "claude_weekly"
			}
		} else {
			continue
		}
		remaining := pct
		usedCopy := used
		windows = append(windows, model.UsageWindow{
			Kind:             kind,
			Group:            group,
			RemainingPercent: &remaining,
			UsedPercent:      &usedCopy,
			ResetDescription: reset,
		})
	}
	if !agyQuotaComplete(windows) {
		return nil, false
	}
	return windows, true
}

// agyQuotaComplete requires at least one window from each independent family
// (gemini and claude_gpt). The AGY CLI may omit 5h windows when the account
// is exhausted or the CLI version doesn't report them, so requiring all 4
// windows would cause valid partial data to be rejected and stale cache to be
// served instead.
func agyQuotaComplete(windows []model.UsageWindow) bool {
	groups := map[string]bool{
		"gemini":     false,
		"claude_gpt": false,
	}
	for _, w := range windows {
		if _, ok := groups[w.Group]; ok {
			groups[w.Group] = true
		}
	}
	for _, present := range groups {
		if !present {
			return false
		}
	}
	return true
}

func formatAgyReset(raw string) string {
	if raw == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	d := time.Until(t)
	if d <= 0 {
		return "Quota available"
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("Refreshes in %dh %dm", hours, mins)
	}
	return fmt.Sprintf("Refreshes in %dm", mins)
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
