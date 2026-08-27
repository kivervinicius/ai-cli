package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ai-manager/internal/conversation"
	"ai-manager/internal/profile"
	"ai-manager/internal/provider/agy"
	"ai-manager/internal/provider/codex"
	"ai-manager/internal/runtime"
	"ai-manager/internal/tui"
)

const version = "0.2.0"

func Run(args []string) error {
	if len(args) == 0 {
		return interactive()
	}

	// Short form: ai agy:personal -- --model ...
	if strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) == 2 && (parts[0] == "agy" || parts[0] == "codex") {
			return runProfile(parts[0], parts[1], trimDashDash(args[1:]))
		}
	}

	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return nil
	case "version", "--version", "-v":
		fmt.Println("ai-manager", version)
		return nil
	case "list", "ls":
		return list()
	case "paths":
		return paths()
	case "doctor":
		return doctor()
	case "add":
		return addCmd(args[1:])
	case "login":
		return loginCmd(args[1:])
	case "inspect":
		return inspectCmd(args[1:])
	case "completion":
		return completionCmd(args[1:])
	case "run":
		return runCmd(args[1:])
	case "resume", "continue":
		return resumeCmd(args[1:])
	case "switch", "swap":
		return switchCmd(args[1:])
	case "use":
		return useCmd(args[1:])
	case "current":
		return currentCmd(args[1:])
	case "status":
		return statusCmd(args[1:])
	case "usage", "quota":
		return usageCmd(args[1:])
	case "remove", "rm":
		return removeCmd(args[1:])
	case "codex", "agy":
		provider := args[0]
		if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
			return runProfile(provider, args[1], trimDashDash(args[2:]))
		}
		name, err := profile.Default(provider)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("no default %s profile; use: ai use %s <profile>", provider, provider)
		}
		return runProfile(provider, name, trimDashDash(args[1:]))
	default:
		if strings.HasPrefix(args[0], "-") {
			cfg, _ := profile.LoadConfig()
			agyDef := cfg.Defaults["agy"]
			codexDef := cfg.Defaults["codex"]
			if agyDef != "" && codexDef == "" {
				return runProfile("agy", agyDef, trimDashDash(args))
			}
			if codexDef != "" && agyDef == "" {
				return runProfile("codex", codexDef, trimDashDash(args))
			}
			if agyDef != "" && codexDef != "" {
				return fmt.Errorf("multiple defaults configured (agy:%s, codex:%s); specify provider, e.g.: ai agy %s or ai codex %s", agyDef, codexDef, strings.Join(args, " "), strings.Join(args, " "))
			}
			return fmt.Errorf("no default profile configured; specify provider, e.g.: ai agy %s or ai codex %s", strings.Join(args, " "), strings.Join(args, " "))
		}
		return fmt.Errorf("unknown command %q; run 'ai help'", args[0])
	}
}

func usage() {
	fmt.Print(`AI Manager v0.2.0 - isolated multi-account launcher for Codex and AGY

Usage:
  ai                              interactive TUI (profiles, accounts, recent conversations)
  ai list                         list profiles with accounts and plans
  ai resume [id] [profile]        resume a recent conversation (with any account)
  ai switch [provider]            quick default account switcher
  ai add codex <name>             create + authenticate a Codex profile
  ai add agy <name>               create + authenticate an AGY profile
  ai login <provider> <name>      start the provider's login flow
  ai inspect <provider> <name>    display non-secret profile metadata
  ai run <provider> <name> -- ... run an external CLI with that profile
  ai codex <name> [-- ...]        short form
  ai agy <name> [-- ...]          short form
  ai codex:<name> [-- ...]        shortest form
  ai agy:<name> [-- ...]          shortest form
  ai use <provider> <name>        set provider default
  ai codex [-- ...]               run default Codex profile
  ai agy [-- ...]                 run default AGY profile
  ai current [provider]            show defaults
  ai status [provider] [name]      local/login status
  ai usage [provider] [name]       real-time quota and /usage monitor
  ai remove <provider> <name>      delete a local profile
  ai doctor                        dependency diagnostics
  ai paths                         show manager storage paths
  ai completion <bash|zsh>         output shell completion script

Examples:
  ai                              # open interactive TUI
  ai resume                       # pick recent conversation & account to continue
  ai switch agy                   # quick switch default AGY account
  ai agy:google-personal -c       # continue latest conversation
  ai agy:google-personal --yolo
  ai codex:openai-work --yolo
`)
}

func addCmd(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ai add <codex|agy> [name] [--no-login]")
	}
	provider := args[0]
	if err := profile.ValidateProvider(provider); err != nil {
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
		fmt.Printf("Profile name for %s: ", provider)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		name = strings.TrimSpace(s)
	}
	p, err := profile.Create(provider, name)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = profile.Delete(provider, name)
		}
	}()
	switch provider {
	case "codex":
		err = codex.Prepare(name)
	case "agy":
		err = agy.Prepare(name)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Created %s profile %q.\n", p.Provider, p.Name)
	// First profile of a provider becomes default.
	if d, _ := profile.Default(provider); d == "" {
		_ = profile.SetDefault(provider, name)
	}
	cleanup = false
	if noLogin {
		return nil
	}
	fmt.Printf("Starting official %s authentication for %q...\n", provider, name)
	if provider == "codex" {
		return codex.Login(name)
	}
	fmt.Println("AGY has no separate login command; its official CLI will start in an isolated keyring. Complete Google Sign-In, then exit AGY when ready.")
	return agy.Login(name)
}

func loginCmd(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: ai login <codex|agy> <name>")
	}
	if !profile.Exists(args[0], args[1]) {
		return fmt.Errorf("profile %s:%s does not exist", args[0], args[1])
	}
	if args[0] == "codex" {
		return codex.Login(args[1])
	}
	if args[0] == "agy" {
		fmt.Println("Launching AGY in this profile. If it is already signed in and you need a different account, run /logout in AGY first, exit, then run this command again.")
		return agy.Login(args[1])
	}
	return profile.ValidateProvider(args[0])
}

func inspectCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai inspect <provider> <name>")
	}
	provider := args[0]
	name := ""
	if strings.Contains(provider, ":") {
		parts := strings.SplitN(provider, ":", 2)
		provider = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	} else {
		return errors.New("usage: ai inspect <provider> <name>")
	}

	if err := profile.ValidateProvider(provider); err != nil {
		return err
	}

	info, err := profile.Inspect(provider, name)
	if err != nil {
		return err
	}

	fmt.Printf("Provider:        %s\n", info.Profile.Provider)
	fmt.Printf("Profile:         %s\n", info.Profile.Name)
	fmt.Printf("Default:         %v\n", info.IsDefault)
	if !info.Profile.CreatedAt.IsZero() {
		fmt.Printf("Created:         %s\n", info.Profile.CreatedAt.Format(time.RFC3339))
	}
	fmt.Printf("Root Path:       %s\n", info.RootPath)
	fmt.Printf("Home Path:       %s\n", info.HomePath)
	fmt.Printf("Config Dir:      %s\n", info.ConfigDir)
	fmt.Printf("Data Dir:        %s\n", info.DataDir)
	fmt.Printf("External Binary: %s\n", info.BinaryPath)
	fmt.Printf("Working Dir:     %s\n", info.CWD)
	fmt.Printf("Process UID/GID: %d / %d\n", info.UID, info.GID)

	if len(info.IsolationVars) > 0 {
		fmt.Println("\nIsolation Environment:")
		var keys []string
		for k := range info.IsolationVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-32s %s\n", k+":", info.IsolationVars[k])
		}
	}

	if len(info.Details) > 0 {
		fmt.Println("\nStatus & Storage:")
		var keys []string
		for k := range info.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-32s %s\n", k+":", info.Details[k])
		}
	}
	return nil
}

func runCmd(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ai run <codex|agy> [name] [--] [args...]")
	}
	provider := args[0]
	name := ""
	rest := args[1:]
	if strings.Contains(provider, ":") {
		parts := strings.SplitN(provider, ":", 2)
		provider = parts[0]
		name = parts[1]
	} else if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		name = rest[0]
		rest = rest[1:]
	}
	if err := profile.ValidateProvider(provider); err != nil {
		return err
	}
	rest = trimDashDash(rest)
	if name == "" {
		var err error
		name, err = profile.Default(provider)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("no default %s profile; use: ai use %s <profile>", provider, provider)
		}
	}
	return runProfile(provider, name, rest)
}

func runProfile(provider, name string, args []string) error {
	if !profile.Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist; create it with 'ai add %s %s'", provider, name, provider, name)
	}
	switch provider {
	case "codex":
		return codex.Run(name, args)
	case "agy":
		return agy.Run(name, args)
	default:
		return profile.ValidateProvider(provider)
	}
}

func useCmd(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: ai use <codex|agy> <name>")
	}
	if err := profile.SetDefault(args[0], args[1]); err != nil {
		return err
	}
	fmt.Printf("Default %s profile: %s\n", args[0], args[1])
	return nil
}

func currentCmd(args []string) error {
	providers := []string{"agy", "codex"}
	if len(args) == 1 {
		providers = []string{args[0]}
	}
	if len(args) > 1 {
		return errors.New("usage: ai current [provider]")
	}
	for _, p := range providers {
		if err := profile.ValidateProvider(p); err != nil {
			return err
		}
		d, err := profile.Default(p)
		if err != nil {
			return err
		}
		if d == "" {
			d = "(none)"
		}
		fmt.Printf("%-5s %s\n", p, d)
	}
	return nil
}

func list() error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	cfg, _ := profile.LoadConfig()
	if len(ps) == 0 {
		fmt.Println("No profiles. Example: ai add agy google-a")
		return nil
	}
	fmt.Printf("%-8s %-22s %-30s %-16s %s\n", "PROVIDER", "PROFILE", "ACCOUNT / EMAIL", "PLAN", "DEFAULT")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		star := ""
		if cfg.Defaults[p.Provider] == p.Name {
			star = "* (default)"
		}
		email := acc.Email
		if len(email) > 28 {
			email = email[:26] + ".."
		}
		fmt.Printf("%-8s %-22s %-30s %-16s %s\n", p.Provider, p.Name, email, acc.Plan, star)
	}
	return nil
}

func resumeCmd(args []string) error {
	if len(args) == 0 {
		cwd, _ := os.Getwd()
		convs := conversation.ListRecent(10, cwd)
		if len(convs) == 0 {
			fmt.Println("No recent conversations found.")
			return nil
		}
		fmt.Println("Recent Conversations:")
		for i, c := range convs {
			prov := strings.ToUpper(c.Provider)
			idPrev := c.ID
			if len(idPrev) > 8 {
				idPrev = idPrev[:8]
			}
			fmt.Printf("  %d) [%s] %-36s (%s)\n", i+1, prov, c.Title, idPrev)
		}
		fmt.Print("Select conversation [1-9] (q to cancel): ")
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
		chosenConv := convs[idx-1]

		ps, err := profile.List()
		if err != nil || len(ps) == 0 {
			return errors.New("no profiles available")
		}
		idPrev := chosenConv.ID
		if len(idPrev) > 8 {
			idPrev = idPrev[:8]
		}
		fmt.Printf("\nResume %q (%s) with account:\n", chosenConv.Title, idPrev)
		for i, p := range ps {
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			fmt.Printf("  %d) %-5s %-16s (%s)\n", i+1, strings.ToUpper(p.Provider), p.Name, acc.Email)
		}
		fmt.Print("Select account [1-9]: ")
		var pInput string
		fmt.Scanln(&pInput)
		pInput = strings.TrimSpace(pInput)
		var pIdx int
		if _, err := fmt.Sscanf(pInput, "%d", &pIdx); err != nil || pIdx < 1 || pIdx > len(ps) {
			return errors.New("invalid profile selection")
		}
		p := ps[pIdx-1]
		return runProfile(p.Provider, p.Name, []string{"--conversation=" + chosenConv.ID})
	}

	convID := args[0]
	profileSpec := ""
	if len(args) >= 2 {
		profileSpec = args[1]
	}

	if profileSpec == "" {
		cfg, _ := profile.LoadConfig()
		if cfg.Defaults["agy"] != "" {
			return runProfile("agy", cfg.Defaults["agy"], []string{"--conversation=" + convID})
		}
		if cfg.Defaults["codex"] != "" {
			return runProfile("codex", cfg.Defaults["codex"], []string{"--conversation=" + convID})
		}
		return errors.New("no default profile configured; specify profile: ai resume <id> <provider:name>")
	}

	provider := "agy"
	name := profileSpec
	if strings.Contains(profileSpec, ":") {
		parts := strings.SplitN(profileSpec, ":", 2)
		provider = parts[0]
		name = parts[1]
	}
	return runProfile(provider, name, []string{"--conversation=" + convID})
}

func switchCmd(args []string) error {
	if len(args) >= 1 && !strings.HasPrefix(args[0], "-") {
		targetProv := ""
		targetName := ""

		if strings.Contains(args[0], ":") {
			parts := strings.SplitN(args[0], ":", 2)
			targetProv = parts[0]
			targetName = parts[1]
		} else if len(args) >= 2 {
			targetProv = args[0]
			targetName = args[1]
		} else {
			// Find by profile name across providers
			nameCandidate := args[0]
			ps, _ := profile.List()
			for _, p := range ps {
				if p.Name == nameCandidate {
					targetProv = p.Provider
					targetName = p.Name
					break
				}
			}
		}

		if targetProv != "" && targetName != "" {
			if !profile.Exists(targetProv, targetName) {
				return fmt.Errorf("profile %s:%s does not exist", targetProv, targetName)
			}
			if err := profile.SetDefault(targetProv, targetName); err != nil {
				return err
			}
			acc := profile.GetAccountInfo(targetProv, targetName)
			fmt.Printf("✓ Alternado com sucesso para %s:%s (%s - %s)\n", strings.ToUpper(targetProv), targetName, acc.Email, acc.Plan)
			fmt.Println("As próximas mensagens no CLI e novas sessões usarão esta conta e sua quota.")
			return nil
		}
	}

	provider := ""
	if len(args) >= 1 {
		provider = args[0]
	}

	ps, err := profile.List()
	if err != nil {
		return err
	}
	var filtered []profile.Profile
	for _, p := range ps {
		if provider == "" || p.Provider == provider {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("no profiles found for provider %q", provider)
	}

	fmt.Println("Select default profile:")
	cfg, _ := profile.LoadConfig()
	for i, p := range filtered {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		isDef := cfg.Defaults[p.Provider] == p.Name
		defStr := ""
		if isDef {
			defStr = "● (current default)"
		}
		fmt.Printf("  %d) %-5s %-16s %-26s %s\n", i+1, strings.ToUpper(p.Provider), p.Name, acc.Email, defStr)
	}
	fmt.Print("Choose profile [1-9] (q to cancel): ")
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "q" || input == "Q" || input == "" {
		return nil
	}
	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(filtered) {
		return errors.New("invalid selection")
	}
	chosen := filtered[idx-1]
	if err := profile.SetDefault(chosen.Provider, chosen.Name); err != nil {
		return err
	}
	acc := profile.GetAccountInfo(chosen.Provider, chosen.Name)
	fmt.Printf("✓ Alternado com sucesso para %s:%s (%s - %s)\n", strings.ToUpper(chosen.Provider), chosen.Name, acc.Email, acc.Plan)
	return nil
}

func statusCmd(args []string) error {
	if len(args) == 0 {
		if err := list(); err != nil {
			return err
		}
		fmt.Println("\nInspect deep metadata and limits with: ai status <provider> <profile>")
		return nil
	}
	provider := args[0]
	if len(args) == 1 {
		ps, err := profile.List()
		if err != nil {
			return err
		}
		for _, p := range ps {
			if p.Provider == provider {
				acc := profile.GetAccountInfo(p.Provider, p.Name)
				fmt.Printf("%s:%s [%s] %s (%s - %s)\n", p.Provider, p.Name, acc.Status, acc.Email, acc.Plan, acc.QuotaSummary)
			}
		}
		return nil
	}
	if len(args) != 2 {
		return errors.New("usage: ai status [provider] [name]")
	}
	name := args[1]
	if !profile.Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}

	acc := profile.GetAccountInfo(provider, name)
	fmt.Printf("=== %s:%s Status & Quota ===\n", strings.ToUpper(provider), name)
	fmt.Printf("Account / Email: %s\n", acc.Email)
	fmt.Printf("Subscription:    %s\n", acc.Plan)
	fmt.Printf("Auth Status:     %s\n", acc.Status)
	if !acc.ExpiresAt.IsZero() {
		fmt.Printf("Expires / Renews: %s\n", acc.ExpiresAt.Format("02/01/2006 (RFC3339: 2006-01-02T15:04:05Z07:00)"))
	}
	if len(acc.Limits) > 0 {
		fmt.Println("\nModel Limits & Quota Capabilities:")
		for _, lim := range acc.Limits {
			fmt.Printf("  • %s\n", lim)
		}
	}
	return nil
}

func usageCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Println("Nenhum perfil configurado.")
		return nil
	}

	if len(args) == 0 {
		fmt.Printf("%-8s %-20s %-30s %-16s %-28s %s\n", "PROVIDER", "PROFILE", "ACCOUNT", "PLAN", "5H LIMIT", "WEEKLY LIMIT")
		for _, p := range ps {
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			email := acc.Email
			if len(email) > 28 {
				email = email[:26] + ".."
			}
			fiveH := fmt.Sprintf("%s %.0f%%", profile.RenderBar(acc.Quota.FiveHour.PercentLeft, 14), acc.Quota.FiveHour.PercentLeft)
			week := fmt.Sprintf("%s %.0f%%", profile.RenderBar(acc.Quota.Weekly.PercentLeft, 14), acc.Quota.Weekly.PercentLeft)
			fmt.Printf("%-8s %-20s %-30s %-16s %-28s %s\n", p.Provider, p.Name, email, acc.Plan, fiveH, week)
		}
		fmt.Println("\nPara ver detalhes completos de uma conta: ai usage <provider> <perfil>")
		return nil
	}

	provider := args[0]
	name := ""
	if strings.Contains(provider, ":") {
		parts := strings.SplitN(provider, ":", 2)
		provider = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	} else {
		return errors.New("usage: ai usage [provider] [name]")
	}

	if !profile.Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}

	acc := profile.GetAccountInfo(provider, name)
	if provider == "codex" {
		fmt.Printf("╭────────────────────────────────────────────────────────────────────────────────╮\n")
		fmt.Printf("│  >_ OpenAI Codex Status & Quota — %-45s│\n", name)
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│ Visit https://chatgpt.com/codex/settings/usage for up-to-date                  │\n")
		fmt.Printf("│ information on rate limits and credits                                         │\n")
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│  Model:                %-56s│\n", acc.Quota.ModelName)
		fmt.Printf("│  Account:              %-56s│\n", acc.Email+" ("+acc.Plan+")")
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│  5h limit:             %-56s│\n", fmt.Sprintf("%s %.0f%% left (%s)", acc.Quota.FiveHour.ProgressBar, acc.Quota.FiveHour.PercentLeft, acc.Quota.FiveHour.ResetTime))
		fmt.Printf("│  Weekly limit:         %-56s│\n", fmt.Sprintf("%s %.0f%% left (%s)", acc.Quota.Weekly.ProgressBar, acc.Quota.Weekly.PercentLeft, acc.Quota.Weekly.ResetTime))
		fmt.Printf("╰────────────────────────────────────────────────────────────────────────────────╯\n")
	} else {
		fmt.Printf("╭────────────────────────────────────────────────────────────────────────────────╮\n")
		fmt.Printf("│  Models & Quota — %-61s│\n", name+" (AGY)")
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│  Account: %-69s│\n", acc.Email+" ("+acc.Plan+")")
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│  GEMINI MODELS (Gemini Flash, Gemini Pro)                                      │\n")
		fmt.Printf("│    Five Hour Limit Remaining:                                                  │\n")
		fmt.Printf("│      %-74s│\n", fmt.Sprintf("%s %.2f%%", acc.Quota.FiveHour.ProgressBar, acc.Quota.FiveHour.PercentLeft))
		fmt.Printf("│      %-74s│\n", fmt.Sprintf("%.0f%% remaining · %s", acc.Quota.FiveHour.PercentLeft, acc.Quota.FiveHour.ResetsIn))
		fmt.Printf("│    Weekly Limit Remaining:                                                     │\n")
		fmt.Printf("│      %-74s│\n", fmt.Sprintf("%s %.2f%%", acc.Quota.Weekly.ProgressBar, acc.Quota.Weekly.PercentLeft))
		fmt.Printf("│      %-74s│\n", fmt.Sprintf("%.0f%% remaining · %s", acc.Quota.Weekly.PercentLeft, acc.Quota.Weekly.ResetsIn))
		fmt.Printf("│                                                                                │\n")
		fmt.Printf("│  CLAUDE AND GPT MODELS (Claude Opus, Claude Sonnet, GPT-OSS)                   │\n")
		fmt.Printf("│    Five Hour Limit: %-59s│\n", fmt.Sprintf("%s %.0f%% (Quota available)", acc.Quota.ClaudeFiveH.ProgressBar, acc.Quota.ClaudeFiveH.PercentLeft))
		fmt.Printf("│    Weekly Limit:    %-59s│\n", fmt.Sprintf("%s %.0f%% (Quota available)", acc.Quota.ClaudeWeek.ProgressBar, acc.Quota.ClaudeWeek.PercentLeft))
		fmt.Printf("╰────────────────────────────────────────────────────────────────────────────────╯\n")
	}
	return nil
}

func removeCmd(args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return errors.New("usage: ai remove <codex|agy> <name> [--yes]")
	}
	provider, name := args[0], args[1]
	if !profile.Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}
	yes := len(args) == 3 && args[2] == "--yes"
	if !yes {
		fmt.Printf("Delete local profile %s:%s and its isolated credentials? [y/N] ", provider, name)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "y" && s != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return profile.Delete(provider, name)
}

func paths() error {
	d, err := profile.DataDir()
	if err != nil {
		return err
	}
	c, err := profile.ConfigDir()
	if err != nil {
		return err
	}
	fmt.Println("data:  ", d)
	fmt.Println("config:", c)
	return nil
}

type DistroInfo struct {
	Name   string
	ID     string
	IDLike string
}

func detectDistro() DistroInfo {
	info := DistroInfo{Name: "Linux (generic)", ID: "linux"}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		data, err = os.ReadFile("/usr/lib/os-release")
	}
	if err != nil {
		return info
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		switch key {
		case "PRETTY_NAME":
			info.Name = val
		case "NAME":
			if info.Name == "Linux (generic)" {
				info.Name = val
			}
		case "ID":
			info.ID = strings.ToLower(val)
		case "ID_LIKE":
			info.IDLike = strings.ToLower(val)
		}
	}
	return info
}

func doctor() error {
	distro := detectDistro()
	fmt.Printf("Distribution: %s\n\n", distro.Name)

	type item struct {
		name     string
		required string
	}
	items := []item{
		{"codex", "for Codex profiles"},
		{"agy", "for AGY profiles"},
		{"dbus-run-session", "for isolated AGY D-Bus sessions"},
		{"gnome-keyring-daemon", "for isolated AGY Secret Service"},
		{"xdg-open", "recommended for browser OAuth"},
	}
	missingPkgs := false
	missingCodex := false
	missingAGY := false

	for _, it := range items {
		if p, err := exec.LookPath(it.name); err == nil {
			fmt.Printf("[ok]      %-22s %s\n", it.name, p)
		} else {
			fmt.Printf("[missing] %-22s %s\n", it.name, it.required)
			if it.name == "codex" {
				missingCodex = true
			} else if it.name == "agy" {
				missingAGY = true
			} else {
				missingPkgs = true
			}
		}
	}

	if binDir, err := runtime.InternalBinDir(); err == nil {
		fmt.Printf("[ok]      %-22s %s\n", "helpers dir", binDir)
	}

	if d, err := profile.DataDir(); err == nil {
		fmt.Printf("[ok]      %-22s %s\n", "data dir", d)
	}
	if c, err := profile.ConfigDir(); err == nil {
		fmt.Printf("[ok]      %-22s %s\n", "config dir", c)
	}

	if missingPkgs || missingCodex || missingAGY {
		fmt.Println("\nInstallation instructions (run manually with appropriate privileges):")

		if missingPkgs {
			fmt.Println("\nPackage manager dependencies for your distribution:")
			id := distro.ID
			idLike := distro.IDLike
			if id == "ubuntu" || id == "debian" || strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
				fmt.Println("  sudo apt install dbus-x11 gnome-keyring xdg-utils")
			} else if id == "fedora" || id == "rhel" || id == "centos" || id == "rocky" || id == "almalinux" || strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
				fmt.Println("  sudo dnf install dbus-daemon gnome-keyring xdg-utils")
			} else if id == "arch" || id == "manjaro" || id == "endeavouros" || strings.Contains(idLike, "arch") {
				fmt.Println("  sudo pacman -S dbus gnome-keyring xdg-utils")
			} else if strings.HasPrefix(id, "opensuse") || id == "sles" || strings.Contains(idLike, "suse") {
				fmt.Println("  sudo zypper install dbus-1 gnome-keyring xdg-utils")
			} else if id == "alpine" {
				fmt.Println("  sudo apk add dbus gnome-keyring xdg-utils")
			} else {
				fmt.Println("  Debian/Ubuntu: sudo apt install dbus-x11 gnome-keyring xdg-utils")
				fmt.Println("  Fedora/RHEL:   sudo dnf install dbus-daemon gnome-keyring xdg-utils")
				fmt.Println("  Arch Linux:    sudo pacman -S dbus gnome-keyring xdg-utils")
				fmt.Println("  openSUSE:      sudo zypper install dbus-1 gnome-keyring xdg-utils")
			}
		}

		if missingCodex || missingAGY {
			fmt.Println("\nOfficial CLI installers:")
			if missingCodex {
				fmt.Println("  Codex: curl -fsSL https://chatgpt.com/codex/install.sh | sh")
			}
			if missingAGY {
				fmt.Println("  AGY:   curl -fsSL https://antigravity.google/cli/install.sh | bash")
			}
		}
	}
	return nil
}

func completionCmd(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ai completion <bash|zsh>")
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletionScript)
		return nil
	case "zsh":
		fmt.Print(zshCompletionScript)
		return nil
	default:
		return fmt.Errorf("unsupported shell %q (supported: bash, zsh)", args[0])
	}
}

const bashCompletionScript = `_ai_completion() {
    local cur prev words cword
    _init_completion -n : || return

    local commands="add login run resume continue switch use current status usage quota inspect remove rm list ls doctor paths completion version help codex agy"
    local providers="codex agy"

    if [[ $cword -eq 1 ]]; then
        if [[ "$cur" == *:* ]]; then
            local profiles
            profiles=$(ai list 2>/dev/null | awk 'NR>1 {print $1":"$2}')
            COMPREPLY=( $(compgen -W "$profiles" -- "$cur") )
            return 0
        fi
        local short_profiles
        short_profiles=$(ai list 2>/dev/null | awk 'NR>1 {print $1":"$2}')
        COMPREPLY=( $(compgen -W "$commands $short_profiles" -- "$cur") )
        return 0
    fi

    case "${words[1]}" in
        add|switch)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$providers" -- "$cur") )
            fi
            ;;
        login|use|inspect|remove|rm|status|usage|quota|run)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$providers" -- "$cur") )
            elif [[ $cword -eq 3 ]]; then
                local prov="${words[2]}"
                local profs
                profs=$(ai list 2>/dev/null | awk -v p="$prov" '$1==p {print $2}')
                COMPREPLY=( $(compgen -W "$profs" -- "$cur") )
            fi
            ;;
        codex)
            local profs
            profs=$(ai list 2>/dev/null | awk '$1=="codex" {print $2}')
            COMPREPLY=( $(compgen -W "$profs" -- "$cur") )
            ;;
        agy)
            local profs
            profs=$(ai list 2>/dev/null | awk '$1=="agy" {print $2}')
            COMPREPLY=( $(compgen -W "$profs" -- "$cur") )
            ;;
        completion)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
            fi
            ;;
    esac
}
complete -F _ai_completion ai
`

const zshCompletionScript = `#compdef ai

_ai() {
    local -a commands
    commands=(
        'add:Create and authenticate a new profile'
        'list:List all configured profiles'
        'ls:List all configured profiles'
        'resume:Resume a recent conversation'
        'switch:Switch default account'
        'usage:Monitor real-time quota and usage'
        'quota:Monitor real-time quota and usage'
        'login:Start provider login flow'
        'run:Run external CLI with explicit profile and args'
        'use:Set default profile for provider'
        'current:Show default profile'
        'status:Show status of profiles'
        'inspect:Show non-secret profile metadata'
        'remove:Delete a profile and its credentials'
        'rm:Delete a profile and its credentials'
        'doctor:Check dependencies and system health'
        'paths:Show manager storage paths'
        'completion:Generate shell completion script'
        'codex:Run Codex profile'
        'agy:Run AGY profile'
        'version:Show version'
        'help:Show help'
    )

    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :->command' \
        '2: :->arg1' \
        '3: :->arg2' \
        '*:: :->args'

    case $state in
        command)
            local -a short_profiles
            short_profiles=(${(f)"$(ai list 2>/dev/null | awk 'NR>1 {print $1":"$2}')"})
            _describe 'commands' commands -- short_profiles
            ;;
        arg1)
            case $line[1] in
                add|switch|login|use|inspect|remove|rm|status|usage|quota|run)
                    _values 'providers' 'codex' 'agy'
                    ;;
                codex)
                    local -a codex_profs
                    codex_profs=(${(f)"$(ai list 2>/dev/null | awk '$1=="codex" {print $2}')"})
                    _values 'profiles' $codex_profs
                    ;;
                agy)
                    local -a agy_profs
                    agy_profs=(${(f)"$(ai list 2>/dev/null | awk '$1=="agy" {print $2}')"})
                    _values 'profiles' $agy_profs
                    ;;
                completion)
                    _values 'shell' 'bash' 'zsh'
                    ;;
            esac
            ;;
        arg2)
            case $line[1] in
                login|use|inspect|remove|rm|status|run)
                    local prov=$line[2]
                    local -a profs
                    profs=(${(f)"$(ai list 2>/dev/null | awk -v p="$prov" '$1==p {print $2}')"})
                    _values 'profiles' $profs
                    ;;
            esac
            ;;
    esac
}

_ai "$@"
`

func interactive() error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		usage()
		return nil
	}

	res, err := tui.ShowMenu()
	if err != nil {
		return err
	}
	if res == nil || res.Action == tui.ActionQuit || res.Action == tui.ActionNone {
		return nil
	}

	switch res.Action {
	case tui.ActionRunProfile:
		return runProfile(res.Provider, res.ProfileName, res.Args)
	case tui.ActionResumeConversation:
		return runProfile(res.Provider, res.ProfileName, []string{"--conversation=" + res.ConversationID})
	case tui.ActionLogin:
		return loginCmd([]string{res.Provider, res.ProfileName})
	default:
		return nil
	}
}

func trimDashDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

// referenced by future worktree support; retained to keep path behavior explicit.
func absCWD() string {
	cwd, _ := os.Getwd()
	a, err := filepath.Abs(cwd)
	if err != nil {
		return cwd
	}
	return a
}

