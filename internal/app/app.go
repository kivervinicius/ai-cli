package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/exitcode"
	"github.com/kivervinicius/ai-cli/internal/core/fallback"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/core/session"
	"github.com/kivervinicius/ai-cli/internal/core/telemetry"
	"github.com/kivervinicius/ai-cli/internal/profile"
	"github.com/kivervinicius/ai-cli/internal/tui"
)

const version = "0.4.0"

var globalRegistry *provider.Registry

func initRegistry() *provider.Registry {
	if globalRegistry != nil {
		return globalRegistry
	}
	reg := provider.NewRegistry()
	_ = reg.Register(codex.New())
	_ = reg.Register(agy.New())
	_ = reg.Register(claude.New())
	_ = reg.Register(opencode.New())
	_ = reg.Register(gemini.New())
	globalRegistry = reg
	return reg
}

func Run(args []string) error {
	initRegistry()

	if len(args) == 0 {
		return interactive()
	}

	// Short form: ai codex:work -- --model ... or ai codex:auto
	if strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		prov := parts[0]
		prof := parts[1]
		if isSupportedProvider(prov) {
			if prof == "auto" {
				prof = ""
			}
			return executeProviderWithSmartSelection(prov, prof, trimDashDash(args[1:]), true)
		}
	}

	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return nil
	case "version", "--version", "-v":
		return versionCmd(args[1:])
	case "providers":
		return providersCmd(args[1:])
	case "profiles", "list", "ls":
		return profilesCmd(args[1:])
	case "paths":
		return paths()
	case "doctor":
		return doctorCmd(args[1:])
	case "security":
		return securityCmd(args[1:])
	case "add":
		return addCmd(args[1:])
	case "remove", "rm":
		return removeCmd(args[1:])
	case "rename":
		return renameCmd(args[1:])
	case "login":
		return loginCmd(args[1:])
	case "logout":
		return logoutCmd(args[1:])
	case "inspect":
		return inspectCmd(args[1:])
	case "completion":
		return completionCmd(args[1:])
	case "run":
		return runCmd(args[1:])
	case "resume", "continue":
		return resumeCmd(args[1:])
	case "switch", "swap", "use":
		return useCmd(args[1:])
	case "current":
		return currentCmd(args[1:])
	case "status":
		return statusCmd(args[1:])
	case "usage", "quota":
		return usageCmd(args[1:])
	case "sessions":
		return sessionsCmd(args[1:])
	case "workspaces":
		return workspacesCmd(args[1:])
	case "bind":
		return bindCmd(args[1:])
	case "unbind":
		return unbindCmd(args[1:])
	case "bindings":
		return bindingsCmd(args[1:])
	case "explain":
		return explainCmd(args[1:])
	case "history":
		return historyCmd(args[1:])
	case "stats":
		return statsCmd(args[1:])
	case "export":
		return exportCmd(args[1:])
	case "issue-report":
		return issueReportCmd(args[1:])
	case "config":
		return configCmd(args[1:])
	case "control", "ui":
		return controlCmd(args[1:])
	case "__control-host":
		return controlHostCmd(args[1:])
	case "codex", "agy", "claude", "opencode", "gemini":
		prov := args[0]
		targetProfile := ""
		rest := args[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			targetProfile = rest[0]
			rest = rest[1:]
		}
		if targetProfile == "auto" {
			targetProfile = ""
		}
		return executeProviderWithSmartSelection(prov, targetProfile, trimDashDash(rest), true)
	default:
		if strings.HasPrefix(args[0], "-") {
			// Auto selection for configured default provider
			cfg, _ := config.LoadConfig()
			for prov, prof := range cfg.Defaults {
				if prof != "" {
					return executeProviderWithSmartSelection(prov, prof, trimDashDash(args), true)
				}
			}
			return fmt.Errorf("no default profile configured; specify provider, e.g.: ai codex")
		}
		return fmt.Errorf("unknown command %q; run 'ai help'", args[0])
	}
}

func isSupportedProvider(p string) bool {
	switch strings.ToLower(p) {
	case "codex", "agy", "claude", "opencode", "gemini":
		return true
	default:
		return false
	}
}

func trimDashDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func interactive() error {
	res, err := tui.ShowMenu()
	if err != nil {
		return err
	}
	if res == nil || res.Action == tui.ActionQuit || res.Action == tui.ActionNone {
		return nil
	}
	switch res.Action {
	case tui.ActionRunProfile:
		return executeProviderWithSmartSelection(res.Provider, res.ProfileName, res.Args, true)
	case tui.ActionResumeConversation:
		return executeResume(res.Provider, res.ProfileName, res.ConversationID, res.Args)
	case tui.ActionLogin:
		return loginCmd([]string{res.Provider, res.ProfileName})
	default:
		return nil
	}
}

func executeProviderWithSmartSelection(provName, explicitProfile string, args []string, allowFallback bool) error {
	reg := initRegistry()
	pAdapter, ok := reg.Get(provName)
	if !ok {
		return fmt.Errorf("unknown provider %q (exit code: %d)", provName, exitcode.ProviderNotFound)
	}

	cfg, _ := config.LoadConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cdTracker)
	exec := fallback.NewExecutor(sel, cdTracker)

	allProfiles, err := profile.List()
	if err != nil {
		return err
	}

	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	for _, p := range allProfiles {
		if p.Provider == provName {
			candidates = append(candidates, p)
			accounts[p.Name] = profile.GetAccountInfo(provName, p.Name)
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no profiles configured for provider %s. Run: ai add %s <name>", provName, provName)
	}

	cwd, _ := os.Getwd()
	ctx := context.Background()

	return exec.RunWithFallback(ctx, provName, cwd, explicitProfile, candidates, accounts, allowFallback, func(p model.Profile) (model.Failure, error) {
		start := time.Now()
		_ = telemetry.LogEvent(telemetry.Event{
			Type:       telemetry.EventSessionStarted,
			ProviderID: provName,
			ProfileID:  p.Name,
			Workspace:  cwd,
		})

		fail, runErr := pAdapter.Run(ctx, p, args)
		dur := time.Since(start)

		if fail.Kind != model.FailureNone && fail.Kind != "" {
			_ = telemetry.LogEvent(telemetry.Event{
				Type:        telemetry.EventRateLimitDetected,
				ProviderID:  provName,
				ProfileID:   p.Name,
				Workspace:   cwd,
				DurationMs:  dur.Milliseconds(),
				FailureKind: fail.Kind,
			})
		}
		return fail, runErr
	})
}

func executeResume(provName, profName, sessionID string, args []string) error {
	reg := initRegistry()
	pAdapter, ok := reg.Get(provName)
	if !ok {
		return fmt.Errorf("provider %q not found", provName)
	}

	if profName == "" {
		cfg, _ := config.LoadConfig()
		profName = cfg.Defaults[provName]
		if profName == "" {
			ps, _ := profile.List()
			for _, p := range ps {
				if p.Provider == provName {
					profName = p.Name
					break
				}
			}
		}
	}

	p := model.Profile{Provider: provName, Name: profName}
	if convProv, ok := pAdapter.(provider.ConversationProvider); ok {
		fail, err := convProv.Resume(context.Background(), p, sessionID, args)
		if err != nil {
			return err
		}
		if fail.Kind != model.FailureNone && fail.Kind != "" {
			return fmt.Errorf("%s", fail.Message)
		}
		return nil
	}

	return fmt.Errorf("provider %s does not support session resume (exit code: %d)", provName, exitcode.ResumeUnsupported)
}

func usage() {
	fmt.Print(`AI CLI Control Plane v0.4.0 - Local Control Plane for AI Coding CLIs

Usage:
  ai                              Open interactive control plane (TUI)
  ai control [subcmd]             AI Control Center & Supervised Runtimes (alias: ai ui)
  ai <provider> [flags]           Launch provider with intelligent account selection
  ai <provider>:<profile> [flags] Launch specific profile (e.g. ai codex:work)
  ai <provider>:auto [flags]      Explicit auto-selection
  ai resume [id] [provider:name]  Resume previous session using provider-native syntax
  ai providers [--json]           List installed providers, versions & capabilities
  ai profiles [--json]            List configured profiles, auth status & priorities
  ai usage [provider] [--json]    Display real-time quota metrics & cache freshness
  ai sessions [search] [--json]   Universal session index across all providers
  ai workspaces [--json]          View workspaces, session history & bindings
  ai bind <provider>:<profile>    Bind current workspace to a preferred profile
  ai unbind <provider>            Unbind current workspace
  ai bindings [--json]            List all active workspace bindings
  ai explain <provider>           Explain account selection decision and scores
  ai doctor [--json]              Deep diagnostics of runtime, keyrings & CLIs
  ai security [profile] [--json]  Audit file sharing and isolation boundary
  ai history [--json]             View local session execution log
  ai stats [--json]               Aggregated statistics (sessions, fallbacks, rate limits)
  ai config <show|validate>       Manage control plane settings
  ai completion <bash|zsh|fish>   Generate shell completion scripts
  ai version [--json]             Display build and platform information
`)
}

func versionCmd(args []string) error {
	if len(args) > 0 && args[0] == "--json" {
		out := map[string]string{
			"version": version,
			"os":      "linux",
			"arch":    "amd64",
			"go":      "1.24.2",
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("ai-cli %s (linux/amd64)\n", version)
	return nil
}

func providersCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	profiles, _ := profile.List()

	profCount := make(map[string]int)
	for _, p := range profiles {
		profCount[p.Provider]++
	}

	if len(args) > 0 && args[0] == "--json" {
		out := make(map[string]interface{})
		for _, p := range reg.List() {
			id := string(p.ID())
			det := detections[id]
			out[id] = map[string]interface{}{
				"name":         p.Name(),
				"installed":    det.Installed,
				"version":      det.Version,
				"binary_path":  det.BinaryPath,
				"profiles":     profCount[id],
				"capabilities": p.Capabilities(),
			}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%-14s %-12s %-24s %-10s %s\n", "PROVIDER", "INSTALLED", "VERSION", "PROFILES", "CAPABILITIES")
	for _, p := range reg.List() {
		id := string(p.ID())
		det := detections[id]
		instStr := "no"
		if det.Installed {
			instStr = "yes"
		}
		verStr := det.Version
		if verStr == "" {
			verStr = "—"
		}
		if len(verStr) > 22 {
			verStr = verStr[:20] + ".."
		}
		caps := p.Capabilities()
		capSummary := fmt.Sprintf("usage:%v resume:%v isolate:%v", caps.Usage, caps.Resume, caps.IsolatedRuntime)
		fmt.Printf("%-14s %-12s %-24s %-10d %s\n", p.Name(), instStr, verStr, profCount[id], capSummary)
	}
	return nil
}

func profilesCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	cfg, _ := config.LoadConfig()

	if len(args) > 0 && args[0] == "--json" {
		type profileJSON struct {
			model.Profile
			Account model.AccountInfo `json:"account"`
			Default bool              `json:"is_default"`
		}
		var list []profileJSON
		for _, p := range ps {
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			list = append(list, profileJSON{
				Profile: p,
				Account: acc,
				Default: cfg.Defaults[p.Provider] == p.Name,
			})
		}
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(ps) == 0 {
		fmt.Println("No profiles configured. Example: ai add codex work")
		return nil
	}

	fmt.Printf("%-10s %-18s %-28s %-16s %-12s %s\n", "PROVIDER", "PROFILE", "ACCOUNT / EMAIL", "PLAN", "STATUS", "DEFAULT")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		star := ""
		if cfg.Defaults[p.Provider] == p.Name {
			star = "★ (default)"
		}
		email := acc.Email
		if email == "" {
			email = "(unauthenticated)"
		}
		if len(email) > 26 {
			email = email[:24] + ".."
		}
		fmt.Printf("%-10s %-18s %-28s %-16s %-12s %s\n", p.Provider, p.Name, email, acc.Plan, acc.Status, star)
	}
	return nil
}

func addCmd(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ai add <provider> [name] [--no-login]")
	}
	providerName := args[0]
	if err := profile.ValidateProvider(providerName); err != nil {
		return err
	}

	noLogin := false
	name := ""
	for _, a := range args[1:] {
		if a == "--no-login" {
			noLogin = true
		} else if name == "" {
			name = a
		} else {
			return fmt.Errorf("unexpected argument %q", a)
		}
	}
	if name == "" {
		fmt.Printf("Profile name for %s: ", providerName)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		name = strings.TrimSpace(s)
	}

	p, err := profile.Create(providerName, name)
	if err != nil {
		return err
	}

	reg := initRegistry()
	pAdapter, ok := reg.Get(providerName)
	if ok {
		if err := pAdapter.Prepare(context.Background(), p); err != nil {
			_ = profile.Delete(providerName, name)
			return err
		}
	}

	fmt.Printf("✓ Created %s profile %q.\n", p.Provider, p.Name)
	if d, _ := config.GetDefaultProfile(providerName); d == "" {
		_ = config.SetDefaultProfile(providerName, name)
	}

	if noLogin {
		return nil
	}
	fmt.Printf("Starting official authentication flow for %s:%s...\n", providerName, name)
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Login(context.Background(), p)
	}
	return nil
}

func removeCmd(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: ai remove <provider> <name> [--yes]")
	}
	prov, name := args[0], args[1]
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	yes := len(args) >= 3 && args[2] == "--yes"
	if !yes {
		fmt.Printf("Delete profile %s:%s and its isolated local storage? [y/N] ", prov, name)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "y" && s != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return profile.Delete(prov, name)
}

func renameCmd(args []string) error {
	if len(args) != 3 {
		return errors.New("usage: ai rename <provider> <old_name> <new_name>")
	}
	return profile.Rename(args[0], args[1], args[2])
}

func loginCmd(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: ai login <provider> <name>")
	}
	prov, name := args[0], args[1]
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	reg := initRegistry()
	pAdapter, ok := reg.Get(prov)
	if !ok {
		return fmt.Errorf("unknown provider %s", prov)
	}
	p := model.Profile{Provider: prov, Name: name}
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Login(context.Background(), p)
	}
	return fmt.Errorf("provider %s does not support login", prov)
}

func logoutCmd(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: ai logout <provider> <name>")
	}
	prov, name := args[0], args[1]
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	reg := initRegistry()
	pAdapter, ok := reg.Get(prov)
	if !ok {
		return fmt.Errorf("unknown provider %s", prov)
	}
	p := model.Profile{Provider: prov, Name: name}
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Logout(context.Background(), p)
	}
	return nil
}

func useCmd(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: ai use <provider> <name>")
	}
	if err := config.SetDefaultProfile(args[0], args[1]); err != nil {
		return err
	}
	fmt.Printf("Default %s profile: %s\n", args[0], args[1])
	return nil
}

func currentCmd(args []string) error {
	providers := []string{"codex", "agy", "claude", "opencode", "gemini"}
	if len(args) >= 1 && args[0] != "--json" {
		providers = []string{args[0]}
	}
	isJSON := len(args) >= 1 && (args[0] == "--json" || (len(args) >= 2 && args[1] == "--json"))

	cwd, _ := os.Getwd()
	cfg, _ := config.LoadConfig()

	if isJSON {
		out := make(map[string]map[string]string)
		for _, p := range providers {
			bound := config.GetBinding(cwd, p)
			def := cfg.Defaults[p]
			out[p] = map[string]string{
				"bound":   bound,
				"default": def,
			}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Workspace: %s\n\n", cwd)
	for _, p := range providers {
		bound := config.GetBinding(cwd, p)
		def := cfg.Defaults[p]
		if def == "" {
			def = "(none)"
		}
		boundStr := ""
		if bound != "" {
			boundStr = fmt.Sprintf(" [bound: %s]", bound)
		}
		fmt.Printf("%-10s %s%s\n", p, def, boundStr)
	}
	return nil
}

func statusCmd(args []string) error {
	if len(args) == 0 {
		return profilesCmd(nil)
	}
	prov := args[0]
	name := ""
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	}

	if name == "" {
		return profilesCmd(nil)
	}

	acc := profile.GetAccountInfo(prov, name)
	fmt.Printf("=== %s:%s Status ===\n", strings.ToUpper(prov), name)
	fmt.Printf("Email:         %s\n", acc.Email)
	fmt.Printf("Plan:          %s\n", acc.Plan)
	fmt.Printf("Status:        %s\n", acc.Status)
	fmt.Printf("Health:        %s\n", acc.Health)
	if !acc.ExpiresAt.IsZero() {
		fmt.Printf("Expires At:    %s\n", acc.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func usageCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}

	isJSON := len(args) > 0 && args[len(args)-1] == "--json"
	if isJSON {
		var list []model.UsageSnapshot
		qEng := quota.NewEngine(5 * time.Minute)
		for _, p := range ps {
			snap, _ := qEng.GetCachedUsage(p.Provider, p.Name)
			list = append(list, snap)
		}
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%-10s %-16s %-24s %-16s %-28s %s\n", "PROVIDER", "PROFILE", "ACCOUNT", "PLAN", "CAPACITY / 5H", "STATUS")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		email := acc.Email
		if email == "" {
			email = "(unauthenticated)"
		}
		if len(email) > 22 {
			email = email[:20] + ".."
		}
		qDetails := profile.GetQuotaDetails(p.Provider, p.Name, acc.Plan, acc.Email)
		fmt.Printf("%-10s %-16s %-24s %-16s %-28s %s\n",
			p.Provider, p.Name, email, acc.Plan, qDetails.FiveHour.ProgressBar, qDetails.Status)
	}
	return nil
}

func sessionsCmd(args []string) error {
	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(50, cwd)

	if len(args) > 0 && args[0] == "search" && len(args) >= 2 {
		q := args[1]
		var filtered []conversation.Conversation
		for _, c := range convs {
			if strings.Contains(strings.ToLower(c.Title), strings.ToLower(q)) ||
				strings.Contains(strings.ToLower(c.ID), strings.ToLower(q)) ||
				strings.Contains(strings.ToLower(c.Workspace), strings.ToLower(q)) {
				filtered = append(filtered, c)
			}
		}
		convs = filtered
	}

	if len(args) > 0 && args[len(args)-1] == "--json" {
		b, _ := json.MarshalIndent(convs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(convs) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Printf("%-10s %-36s %-32s %s\n", "PROVIDER", "SESSION ID", "TITLE", "WORKSPACE")
	for _, c := range convs {
		title := c.Title
		if len(title) > 30 {
			title = title[:28] + ".."
		}
		ws := c.Workspace
		if len(ws) > 30 {
			ws = ws[:28] + ".."
		}
		fmt.Printf("%-10s %-36s %-32s %s\n", c.Provider, c.ID, title, ws)
	}
	return nil
}

func workspacesCmd(args []string) error {
	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(100, cwd)
	cfg, _ := config.LoadConfig()

	var sessions []model.Session
	for _, c := range convs {
		sessions = append(sessions, model.Session{
			ProviderID: c.Provider,
			ID:         c.ID,
			Title:      c.Title,
			Workspace:  c.Workspace,
			UpdatedAt:  c.LastModified,
		})
	}

	store := session.NewStore()
	wsList := store.GroupByWorkspace(sessions, cfg)

	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(wsList, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Active Workspaces (%d discovered):\n\n", len(wsList))
	for _, ws := range wsList {
		fmt.Printf("📁 %s\n", ws.Path)
		if len(ws.Bindings) > 0 {
			fmt.Printf("   Bindings: %+v\n", ws.Bindings)
		}
		fmt.Printf("   Sessions: %d\n\n", len(ws.Sessions))
	}
	return nil
}

func bindCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: ai bind <provider>:<profile>")
	}
	parts := strings.SplitN(args[0], ":", 2)
	if len(parts) != 2 {
		return errors.New("usage: ai bind <provider>:<profile>")
	}
	prov, prof := parts[0], parts[1]
	cwd, _ := os.Getwd()
	if err := config.BindWorkspace(cwd, prov, prof); err != nil {
		return err
	}
	fmt.Printf("✓ Bound workspace %s to %s:%s\n", cwd, prov, prof)
	return nil
}

func unbindCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: ai unbind <provider>")
	}
	cwd, _ := os.Getwd()
	if err := config.UnbindWorkspace(cwd, args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ Unbound provider %s from workspace %s\n", args[0], cwd)
	return nil
}

func bindingsCmd(args []string) error {
	cfg, _ := config.LoadConfig()
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(cfg.Bindings, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(cfg.Bindings) == 0 {
		fmt.Println("No workspace bindings configured.")
		return nil
	}
	for ws, b := range cfg.Bindings {
		fmt.Printf("%s:\n", ws)
		for p, prof := range b {
			fmt.Printf("  • %-10s -> %s\n", p, prof)
		}
	}
	return nil
}

func explainCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: ai explain <provider>")
	}
	prov := args[0]
	cfg, _ := config.LoadConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cdTracker)

	ps, _ := profile.List()
	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	for _, p := range ps {
		if p.Provider == prov {
			candidates = append(candidates, p)
			accounts[p.Name] = profile.GetAccountInfo(prov, p.Name)
		}
	}

	cwd, _ := os.Getwd()
	fmt.Print(sel.ExplainSelection(context.Background(), prov, cwd, candidates, accounts))
	return nil
}

func doctorCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	ps, _ := profile.List()

	if len(args) > 0 && args[0] == "--json" {
		out := map[string]interface{}{
			"detections": detections,
			"profiles":   ps,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Println("=== AI CLI Diagnostics ===")
	for id, det := range detections {
		if det.Installed {
			fmt.Printf("✓ %-12s installed (%s) at %s\n", id, det.Version, det.BinaryPath)
		} else {
			fmt.Printf("✗ %-12s not installed (%s)\n", id, det.Error)
		}
	}

	fmt.Println("\n=== Profile & Auth Status ===")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		if acc.Authenticated {
			fmt.Printf("✓ %s:%s authenticated (%s)\n", p.Provider, p.Name, acc.Email)
		} else {
			fmt.Printf("⚠ %s:%s unauthenticated (%s)\n", p.Provider, p.Name, acc.Status)
		}
	}
	return nil
}

func securityCmd(args []string) error {
	cfg, _ := config.LoadConfig()
	ps, _ := profile.List()
	hostHome := security.FindHostHome()

	if len(args) > 0 && args[0] == "--json" {
		var audits []security.SecurityAudit
		for _, p := range ps {
			home, _ := config.ProfileHome(p.Provider, p.Name)
			audits = append(audits, security.AuditProfile(p.Provider, p.Name, home, hostHome, cfg.IsolationPreset))
		}
		b, _ := json.MarshalIndent(audits, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Isolation Preset: %s\n\n", cfg.IsolationPreset)
	for _, p := range ps {
		home, _ := config.ProfileHome(p.Provider, p.Name)
		audit := security.AuditProfile(p.Provider, p.Name, home, hostHome, cfg.IsolationPreset)
		fmt.Printf("Profile: %s\n", audit.Profile)
		fmt.Println("  Shared:")
		for _, s := range audit.Shared {
			fmt.Printf("    ✓ %s\n", s)
		}
		fmt.Println("  Protected / Not shared:")
		for _, pt := range audit.Protected {
			fmt.Printf("    ✓ %s\n", pt)
		}
		if len(audit.Warnings) > 0 {
			fmt.Println("  Warnings:")
			for _, w := range audit.Warnings {
				fmt.Printf("    ⚠ %s\n", w)
			}
		}
		fmt.Println()
	}
	return nil
}

func historyCmd(args []string) error {
	events, err := telemetry.ReadRecentEvents(50)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(events, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(events) == 0 {
		fmt.Println("No recent history recorded.")
		return nil
	}
	for _, ev := range events {
		fmt.Printf("%s  %-22s  %s:%s  %s\n",
			ev.Timestamp.Format("15:04:05"), ev.Type, ev.ProviderID, ev.ProfileID, ev.Reason)
	}
	return nil
}

func statsCmd(args []string) error {
	s, err := telemetry.ComputeStats(7 * 24 * time.Hour)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("Local Statistics (Last 7 Days):\n\n")
	fmt.Printf("Total Sessions:   %d\n", s.TotalSessions)
	fmt.Printf("Total Fallbacks:  %d\n", s.TotalFallbacks)
	fmt.Printf("Rate Limits:      %d\n\n", s.TotalRateLimits)
	fmt.Println("Sessions by Provider:")
	for p, count := range s.ByProvider {
		fmt.Printf("  • %-12s %d\n", p, count)
	}
	return nil
}

func exportCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	ps, _ := profile.List()
	cfg, _ := config.LoadConfig()

	exportData := map[string]interface{}{
		"version":    version,
		"os":         "linux",
		"detections": detections,
		"profiles":   ps,
		"config":     cfg,
	}

	b, _ := json.MarshalIndent(exportData, "", "  ")
	sanitized := security.Redact(string(b))
	fmt.Println(sanitized)
	return nil
}

func issueReportCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)

	var sb strings.Builder
	sb.WriteString("### Environment Diagnostics\n\n")
	sb.WriteString(fmt.Sprintf("- AI CLI Version: `%s`\n", version))
	sb.WriteString("- OS: `linux`\n\n")
	sb.WriteString("### Installed Providers\n\n")
	for id, det := range detections {
		sb.WriteString(fmt.Sprintf("- **%s**: Installed=%v, Version=`%s`\n", id, det.Installed, det.Version))
	}
	fmt.Println(security.Redact(sb.String()))
	return nil
}

func configCmd(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if args[0] == "validate" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		issues := cfg.Validate()
		if len(issues) == 0 {
			fmt.Println("✓ Configuration is valid.")
			return nil
		}
		fmt.Println("Configuration issues found:")
		for _, is := range issues {
			fmt.Printf("  ⚠ %s\n", is)
		}
		return nil
	}
	return errors.New("usage: ai config <show|validate>")
}

func inspectCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai inspect <provider> <name>")
	}
	prov := args[0]
	name := ""
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	}
	info, err := profile.Inspect(prov, name)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(b))
	return nil
}

func runCmd(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ai run <provider> [profile] [--] [args...]")
	}
	prov := args[0]
	prof := ""
	rest := args[1:]
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		prof = parts[1]
	} else if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		prof = rest[0]
		rest = rest[1:]
	}
	return executeProviderWithSmartSelection(prov, prof, trimDashDash(rest), true)
}

func resumeCmd(args []string) error {
	if len(args) == 0 {
		cwd, _ := os.Getwd()
		convs := conversation.ListRecent(10, cwd)
		if len(convs) == 0 {
			fmt.Println("No recent sessions found.")
			return nil
		}
		fmt.Println("Recent Sessions:")
		for i, c := range convs {
			fmt.Printf("  %d) [%s] %-36s (%s)\n", i+1, strings.ToUpper(c.Provider), c.Title, c.ID[:8])
		}
		fmt.Print("Select session [1-9] (q to cancel): ")
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "q" || input == "Q" || input == "" {
			return nil
		}
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(convs) {
			return errors.New("invalid selection")
		}
		chosen := convs[idx-1]
		return executeResume(chosen.Provider, "", chosen.ID, nil)
	}

	sessionID := args[0]
	prov := "codex"
	prof := ""
	if len(args) >= 2 {
		target := args[1]
		if strings.Contains(target, ":") {
			parts := strings.SplitN(target, ":", 2)
			prov = parts[0]
			prof = parts[1]
		} else {
			prof = target
		}
	}
	return executeResume(prov, prof, sessionID, nil)
}

func paths() error {
	d, err := config.DataDir()
	if err != nil {
		return err
	}
	c, err := config.ConfigDir()
	if err != nil {
		return err
	}
	s, err := config.StateDir()
	if err != nil {
		return err
	}
	fmt.Println("data:  ", d)
	fmt.Println("config:", c)
	fmt.Println("state: ", s)
	return nil
}

func completionCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai completion <bash|zsh|fish|powershell>")
	}
	switch args[0] {
	case "bash":
		fmt.Print(`_ai_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="control ui providers profiles usage sessions workspaces bind unbind explain doctor security history stats config completion version codex agy claude opencode gemini"
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _ai_completion ai
`)
	case "zsh":
		fmt.Print(`#compdef ai
_ai() {
    local -a commands
    commands=(
        'control:AI Control Center & Supervised Runtimes'
        'ui:Interactive Control Center'
        'providers:List AI providers'
        'profiles:List configured profiles'
        'usage:Display quota and rate limits'
        'sessions:Universal session index'
        'doctor:Diagnostics and health checks'
        'security:Security and isolation audit'
    )
    _describe 'command' commands
}
_ai "$@"
`)
	case "fish":
		fmt.Print(`complete -c ai -f -a "control ui providers profiles usage sessions doctor security history stats"
`)
	case "powershell", "pwsh":
		fmt.Print(`Register-ArgumentCompleter -Native -CommandName ai -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @(
        'control', 'ui', 'providers', 'profiles', 'usage', 'sessions', 'workspaces',
        'bind', 'unbind', 'explain', 'doctor', 'security',
        'history', 'stats', 'config', 'completion', 'version',
        'codex', 'agy', 'claude', 'opencode', 'gemini'
    )
    $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`)
	default:
		return fmt.Errorf("unsupported shell %q. Supported: bash, zsh, fish, powershell", args[0])
	}
	return nil
}
