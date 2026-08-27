package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"ai-manager/internal/conversation"
	"ai-manager/internal/profile"
)

type ActionType int

const (
	ActionNone ActionType = iota
	ActionRunProfile
	ActionResumeConversation
	ActionSetDefault
	ActionLogin
	ActionQuit
)

type SelectionResult struct {
	Action         ActionType
	Provider       string
	ProfileName    string
	ConversationID string
	Args           []string
}

type ProfileItem struct {
	Profile profile.Profile
	Account profile.AccountInfo
	Default bool
}

const BoxWidth = 96

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func visibleLen(s string) int {
	clean := ansiRegex.ReplaceAllString(s, "")
	return utf8.RuneCountInString(clean)
}

func padRight(s string, width int) string {
	vLen := visibleLen(s)
	if vLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vLen)
}

func drawBorderTop(b *strings.Builder, color string, width int) {
	b.WriteString(color + "┌" + strings.Repeat("─", width-2) + "┐\x1b[0m\n")
}

func drawBorderMid(b *strings.Builder, color string, width int) {
	b.WriteString(color + "├" + strings.Repeat("─", width-2) + "┤\x1b[0m\n")
}

func drawBorderBottom(b *strings.Builder, color string, width int) {
	b.WriteString(color + "└" + strings.Repeat("─", width-2) + "┘\x1b[0m\n")
}

func drawRow(b *strings.Builder, content string, borderColor string, width int) {
	b.WriteString(borderColor + "│\x1b[0m ")
	b.WriteString(padRight(content, width-4))
	b.WriteString(" " + borderColor + "│\x1b[0m\n")
}

// ShowMenu runs the interactive terminal user interface.
func ShowMenu() (*SelectionResult, error) {
	if !isTTY() {
		return showFallbackMenu()
	}

	profiles, err := profile.List()
	if err != nil {
		return nil, err
	}
	cfg, _ := profile.LoadConfig()

	var profileItems []ProfileItem
	for _, p := range profiles {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		isDef := cfg.Defaults[p.Provider] == p.Name
		profileItems = append(profileItems, ProfileItem{
			Profile: p,
			Account: acc,
			Default: isDef,
		})
	}

	cwd, _ := os.Getwd()
	conversations := conversation.ListRecent(15, cwd)

	currentTab := 0 // 0 = Profiles, 1 = Conversations
	selectedProfile := 0
	selectedConv := 0
	statusMsg := ""

	// Set raw mode
	oldState, err := setRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer restore(int(os.Stdin.Fd()), oldState)
	}

	// Hide cursor on start, restore on exit
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h\n")

	for {
		// Render
		renderScreen(currentTab, profileItems, conversations, selectedProfile, selectedConv, statusMsg)
		statusMsg = ""

		// Read key
		key := readKey()
		switch key {
		case "QUIT", "CTRL_C", "ESC":
			return &SelectionResult{Action: ActionQuit}, nil
		case "TAB", "RIGHT":
			currentTab = (currentTab + 1) % 2
		case "LEFT":
			currentTab = (currentTab + 1) % 2
		case "UP":
			if currentTab == 0 && len(profileItems) > 0 {
				selectedProfile = (selectedProfile - 1 + len(profileItems)) % len(profileItems)
			} else if currentTab == 1 && len(conversations) > 0 {
				selectedConv = (selectedConv - 1 + len(conversations)) % len(conversations)
			}
		case "DOWN":
			if currentTab == 0 && len(profileItems) > 0 {
				selectedProfile = (selectedProfile + 1) % len(profileItems)
			} else if currentTab == 1 && len(conversations) > 0 {
				selectedConv = (selectedConv + 1) % len(conversations)
			}
		case "ENTER":
			if currentTab == 0 && len(profileItems) > 0 {
				p := profileItems[selectedProfile]
				return &SelectionResult{
					Action:      ActionRunProfile,
					Provider:    p.Profile.Provider,
					ProfileName: p.Profile.Name,
				}, nil
			} else if currentTab == 1 && len(conversations) > 0 {
				conv := conversations[selectedConv]
				chosenProfile := showProfileModalForResume(conv, profileItems)
				if chosenProfile != nil {
					return &SelectionResult{
						Action:         ActionResumeConversation,
						Provider:       chosenProfile.Profile.Provider,
						ProfileName:    chosenProfile.Profile.Name,
						ConversationID: conv.ID,
					}, nil
				}
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(key[0] - '1')
			if currentTab == 0 && idx < len(profileItems) {
				p := profileItems[idx]
				return &SelectionResult{
					Action:      ActionRunProfile,
					Provider:    p.Profile.Provider,
					ProfileName: p.Profile.Name,
				}, nil
			} else if currentTab == 1 && idx < len(conversations) {
				conv := conversations[idx]
				chosenProfile := showProfileModalForResume(conv, profileItems)
				if chosenProfile != nil {
					return &SelectionResult{
						Action:         ActionResumeConversation,
						Provider:       chosenProfile.Profile.Provider,
						ProfileName:    chosenProfile.Profile.Name,
						ConversationID: conv.ID,
					}, nil
				}
			}
		case "c", "C":
			// Continue latest conversation with selected/default profile
			if len(profileItems) > 0 && len(conversations) > 0 {
				p := profileItems[selectedProfile]
				conv := conversations[0]
				return &SelectionResult{
					Action:         ActionResumeConversation,
					Provider:       p.Profile.Provider,
					ProfileName:    p.Profile.Name,
					ConversationID: conv.ID,
				}, nil
			}
		case "d", "D":
			// Set as default
			if currentTab == 0 && len(profileItems) > 0 {
				p := profileItems[selectedProfile]
				_ = profile.SetDefault(p.Profile.Provider, p.Profile.Name)
				cfg, _ = profile.LoadConfig()
				for i := range profileItems {
					profileItems[i].Default = cfg.Defaults[profileItems[i].Profile.Provider] == profileItems[i].Profile.Name
				}
				statusMsg = fmt.Sprintf("✓ Definido %s:%s como perfil padrão.", p.Profile.Provider, p.Profile.Name)
			}
		case "s", "S":
			// Show Quota / Limits modal
			if currentTab == 0 && len(profileItems) > 0 {
				showQuotaModal(profileItems[selectedProfile])
			}
		case "l", "L":
			// Login
			if currentTab == 0 && len(profileItems) > 0 {
				p := profileItems[selectedProfile]
				return &SelectionResult{
					Action:      ActionLogin,
					Provider:    p.Profile.Provider,
					ProfileName: p.Profile.Name,
				}, nil
			}
		}
	}
}

func renderScreen(tab int, profiles []ProfileItem, convs []conversation.Conversation, selProf, selConv int, status string) {
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J") // Clear screen and move to top-left

	cwd, _ := os.Getwd()
	projName := filepath.Base(cwd)
	if len(projName) > 28 {
		projName = projName[:26] + ".."
	}

	drawBorderTop(&b, "\x1b[1;36m", BoxWidth)

	// Title Row
	projLabel := "\x1b[90m[Projeto: " + projName + "]\x1b[0m"
	titleLeft := "\x1b[1;37mAI Manager\x1b[0m \x1b[90mv0.2.0\x1b[0m"
	spaces := BoxWidth - 4 - visibleLen(titleLeft) - visibleLen(projLabel)
	if spaces < 2 {
		spaces = 2
	}
	drawRow(&b, titleLeft+strings.Repeat(" ", spaces)+projLabel, "\x1b[1;36m", BoxWidth)

	// Tabs Row
	tab1 := "\x1b[90m[ 1. Contas & Perfis ]\x1b[0m"
	tab2 := "\x1b[90m[ 2. Conversas Recentes ]\x1b[0m"
	if tab == 0 {
		tab1 = "\x1b[1;7;36m [ 1. Contas & Perfis ] \x1b[0m"
	} else {
		tab2 = "\x1b[1;7;36m [ 2. Conversas Recentes ] \x1b[0m"
	}
	drawRow(&b, fmt.Sprintf(" Abas:  %s   %s", tab1, tab2), "\x1b[1;36m", BoxWidth)

	drawBorderMid(&b, "\x1b[1;36m", BoxWidth)

	if tab == 0 {
		// Profiles Tab
		drawRow(&b, " \x1b[1;33mCONTAS & PERFIS CONFIGURADOS:\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)

		if len(profiles) == 0 {
			drawRow(&b, "   Nenhum perfil configurado. Crie um com: ai add agy <nome>", "\x1b[1;36m", BoxWidth)
		} else {
			for i, p := range profiles {
				pointer := "  "
				color := "\x1b[0m"
				if i == selProf {
					pointer = "\x1b[1;32m> \x1b[0m"
					color = "\x1b[1;37m"
				}

				provBadge := "\x1b[1;34mAGY  \x1b[0m"
				if p.Profile.Provider == "codex" {
					provBadge = "\x1b[1;32mCODEX\x1b[0m"
				}

				defBadge := ""
				if p.Default {
					defBadge = "\x1b[1;33m★ (Padrão)\x1b[0m"
				}

				email := p.Account.Email
				if len(email) > 28 {
					email = email[:26] + ".."
				}

				name := p.Profile.Name
				if len(name) > 20 {
					name = name[:18] + ".."
				}

				plan := p.Account.Plan
				if plan == "" || plan == "Unknown" {
					plan = "-"
				}
				planFormatted := fmt.Sprintf("\x1b[35m%-15s\x1b[0m", plan)

				line := fmt.Sprintf("%s[%d] %s %s%-20s\x1b[0m \x1b[36m%-28s\x1b[0m %s %s", pointer, i+1, provBadge, color, name, email, planFormatted, defBadge)
				drawRow(&b, line, "\x1b[1;36m", BoxWidth)
			}
		}

		for i := len(profiles); i < 6; i++ {
			drawRow(&b, "", "\x1b[1;36m", BoxWidth)
		}

		drawBorderMid(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;37m[Enter]\x1b[0m Iniciar   \x1b[1;37m[c]\x1b[0m Continuar Última   \x1b[1;37m[d]\x1b[0m Tornar Padrão   \x1b[1;37m[s]\x1b[0m Ver Quotas", "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;37m[Tab]\x1b[0m Conversas    \x1b[1;37m[↑/↓]\x1b[0m Navegar            \x1b[1;37m[1-9]\x1b[0m Seleção Rápida   \x1b[1;37m[q]\x1b[0m Sair", "\x1b[1;36m", BoxWidth)
	} else {
		// Recent Conversations Tab
		drawRow(&b, " \x1b[1;33mCONVERSAS RECENTES (Retomar com qualquer conta):\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)

		if len(convs) == 0 {
			drawRow(&b, "   Nenhuma conversa recente encontrada.", "\x1b[1;36m", BoxWidth)
		} else {
			for i, c := range convs {
				if i >= 6 {
					break
				}
				pointer := "  "
				color := "\x1b[0m"
				if i == selConv {
					pointer = "\x1b[1;32m> \x1b[0m"
					color = "\x1b[1;37m"
				}

				title := c.Title
				if len(title) > 48 {
					title = title[:45] + "..."
				}

				timeAgo := formatTimeAgo(c.LastModified)
				idPrev := c.ID
				if len(idPrev) > 8 {
					idPrev = idPrev[:8]
				}

				line := fmt.Sprintf("%s[%d] %s%-48s\x1b[0m \x1b[90m%s\x1b[0m  \x1b[33m(%s)\x1b[0m", pointer, i+1, color, title, idPrev, timeAgo)
				drawRow(&b, line, "\x1b[1;36m", BoxWidth)
			}
		}

		for i := len(convs); i < 6; i++ {
			drawRow(&b, "", "\x1b[1;36m", BoxWidth)
		}

		drawBorderMid(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;37m[Enter]\x1b[0m Retomar com Outra Conta...    \x1b[1;37m[Tab]\x1b[0m Voltar para Perfis", "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;37m[↑/↓]\x1b[0m Navegar                         \x1b[1;37m[1-9]\x1b[0m Seleção Rápida   \x1b[1;37m[q]\x1b[0m Sair", "\x1b[1;36m", BoxWidth)
	}

	if status != "" {
		drawBorderMid(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;32m"+status+"\x1b[0m", "\x1b[1;36m", BoxWidth)
	}
	drawBorderBottom(&b, "\x1b[1;36m", BoxWidth)

	fmt.Print(b.String())
}

func showQuotaModal(p ProfileItem) {
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J")

	if p.Profile.Provider == "codex" {
		drawBorderTop(&b, "\x1b[1;32m", BoxWidth)
		drawRow(&b, " \x1b[1;37m>_ OpenAI Codex Status & Quota — "+p.Profile.Name+"\x1b[0m", "\x1b[1;32m", BoxWidth)
		drawBorderMid(&b, "\x1b[1;32m", BoxWidth)
		drawRow(&b, "", "\x1b[1;32m", BoxWidth)
		drawRow(&b, " \x1b[90mVisit https://chatgpt.com/codex/settings/usage for up-to-date information\x1b[0m", "\x1b[1;32m", BoxWidth)
		drawRow(&b, "", "\x1b[1;32m", BoxWidth)
		drawRow(&b, fmt.Sprintf("  \x1b[1;37mModel:\x1b[0m                \x1b[36m%s\x1b[0m", p.Account.Quota.ModelName), "\x1b[1;32m", BoxWidth)
		drawRow(&b, fmt.Sprintf("  \x1b[1;37mAccount:\x1b[0m              \x1b[36m%s\x1b[0m \x1b[35m(%s)\x1b[0m", p.Account.Email, p.Account.Plan), "\x1b[1;32m", BoxWidth)
		cwd, _ := os.Getwd()
		drawRow(&b, fmt.Sprintf("  \x1b[1;37mDirectory:\x1b[0m            \x1b[90m%s\x1b[0m", cwd), "\x1b[1;32m", BoxWidth)
		drawRow(&b, "", "\x1b[1;32m", BoxWidth)

		fiveH := fmt.Sprintf("  \x1b[1;37m5h limit:\x1b[0m             \x1b[32m%s\x1b[0m  %.0f%% left \x1b[90m(%s)\x1b[0m", p.Account.Quota.FiveHour.ProgressBar, p.Account.Quota.FiveHour.PercentLeft, p.Account.Quota.FiveHour.ResetTime)
		drawRow(&b, fiveH, "\x1b[1;32m", BoxWidth)

		week := fmt.Sprintf("  \x1b[1;37mWeekly limit:\x1b[0m         \x1b[32m%s\x1b[0m  %.0f%% left \x1b[90m(%s)\x1b[0m", p.Account.Quota.Weekly.ProgressBar, p.Account.Quota.Weekly.PercentLeft, p.Account.Quota.Weekly.ResetTime)
		drawRow(&b, week, "\x1b[1;32m", BoxWidth)

		drawRow(&b, "", "\x1b[1;32m", BoxWidth)
		drawBorderMid(&b, "\x1b[1;32m", BoxWidth)
		drawRow(&b, "  \x1b[1;37m[Pressione qualquer tecla para voltar]\x1b[0m", "\x1b[1;32m", BoxWidth)
		drawBorderBottom(&b, "\x1b[1;32m", BoxWidth)
	} else {
		drawBorderTop(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, " \x1b[1;37mModels & Quota — "+p.Profile.Name+" (AGY)\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawBorderMid(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("  \x1b[1;37mAccount:\x1b[0m \x1b[36m%s\x1b[0m \x1b[35m(%s)\x1b[0m", p.Account.Email, p.Account.Plan), "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)

		drawRow(&b, "  \x1b[1;33mGEMINI MODELS (Gemini Flash, Gemini Pro)\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, "    \x1b[1;37mFive Hour Limit Remaining:\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("      \x1b[36m%s\x1b[0m  %.2f%%", p.Account.Quota.FiveHour.ProgressBar, p.Account.Quota.FiveHour.PercentLeft), "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("      \x1b[90m%.0f%% remaining · %s\x1b[0m", p.Account.Quota.FiveHour.PercentLeft, p.Account.Quota.FiveHour.ResetsIn), "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)

		drawRow(&b, "    \x1b[1;37mWeekly Limit Remaining:\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("      \x1b[36m%s\x1b[0m  %.2f%%", p.Account.Quota.Weekly.ProgressBar, p.Account.Quota.Weekly.PercentLeft), "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("      \x1b[90m%.0f%% remaining · %s\x1b[0m", p.Account.Quota.Weekly.PercentLeft, p.Account.Quota.Weekly.ResetsIn), "\x1b[1;36m", BoxWidth)
		drawRow(&b, "", "\x1b[1;36m", BoxWidth)

		drawRow(&b, "  \x1b[1;33mCLAUDE AND GPT MODELS (Claude Opus, Claude Sonnet, GPT-OSS)\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("    Five Hour Limit: \x1b[36m%s\x1b[0m  %.0f%% \x1b[90m(Quota available)\x1b[0m", p.Account.Quota.ClaudeFiveH.ProgressBar, p.Account.Quota.ClaudeFiveH.PercentLeft), "\x1b[1;36m", BoxWidth)
		drawRow(&b, fmt.Sprintf("    Weekly Limit:    \x1b[36m%s\x1b[0m  %.0f%% \x1b[90m(Quota available)\x1b[0m", p.Account.Quota.ClaudeWeek.ProgressBar, p.Account.Quota.ClaudeWeek.PercentLeft), "\x1b[1;36m", BoxWidth)

		drawRow(&b, "", "\x1b[1;36m", BoxWidth)
		drawBorderMid(&b, "\x1b[1;36m", BoxWidth)
		drawRow(&b, "  \x1b[1;37m[Pressione qualquer tecla para voltar]\x1b[0m", "\x1b[1;36m", BoxWidth)
		drawBorderBottom(&b, "\x1b[1;36m", BoxWidth)
	}

	fmt.Print(b.String())
	_ = readKey()
}

func showProfileModalForResume(conv conversation.Conversation, profiles []ProfileItem) *ProfileItem {
	if len(profiles) == 0 {
		return nil
	}

	selected := 0
	for {
		var b strings.Builder
		b.WriteString("\x1b[H\x1b[2J")
		drawBorderTop(&b, "\x1b[1;35m", BoxWidth)

		title := conv.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		idPrev := conv.ID
		if len(idPrev) > 8 {
			idPrev = idPrev[:8]
		}
		header := fmt.Sprintf(" Continuar \"%s\" (%s) com qual conta?", title, idPrev)
		drawRow(&b, "\x1b[1;37m"+header+"\x1b[0m", "\x1b[1;35m", BoxWidth)

		drawBorderMid(&b, "\x1b[1;35m", BoxWidth)
		drawRow(&b, "", "\x1b[1;35m", BoxWidth)

		for i, p := range profiles {
			pointer := "  "
			color := "\x1b[0m"
			if i == selected {
				pointer = "\x1b[1;32m> \x1b[0m"
				color = "\x1b[1;37m"
			}
			provBadge := "\x1b[1;34mAGY  \x1b[0m"
			if p.Profile.Provider == "codex" {
				provBadge = "\x1b[1;32mCODEX\x1b[0m"
			}
			email := p.Account.Email
			if len(email) > 28 {
				email = email[:26] + ".."
			}
			name := p.Profile.Name
			if len(name) > 20 {
				name = name[:18] + ".."
			}
			plan := p.Account.Plan
			if plan == "" || plan == "Unknown" {
				plan = "-"
			}
			line := fmt.Sprintf("%s[%d] %s %s%-20s\x1b[0m \x1b[36m%-28s\x1b[0m \x1b[35m%-15s\x1b[0m", pointer, i+1, provBadge, color, name, email, plan)
			drawRow(&b, line, "\x1b[1;35m", BoxWidth)
		}

		drawRow(&b, "", "\x1b[1;35m", BoxWidth)
		drawBorderMid(&b, "\x1b[1;35m", BoxWidth)
		drawRow(&b, " \x1b[1;37m[Enter/1-9]\x1b[0m Iniciar nesta conta    \x1b[1;37m[Esc/q]\x1b[0m Voltar", "\x1b[1;35m", BoxWidth)
		drawBorderBottom(&b, "\x1b[1;35m", BoxWidth)

		fmt.Print(b.String())

		key := readKey()
		switch key {
		case "ESC", "q", "Q":
			return nil
		case "UP":
			selected = (selected - 1 + len(profiles)) % len(profiles)
		case "DOWN":
			selected = (selected + 1) % len(profiles)
		case "ENTER":
			return &profiles[selected]
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(key[0] - '1')
			if idx < len(profiles) {
				return &profiles[idx]
			}
		}
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "desconhecido"
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return "agora"
	}
	if diff < time.Hour {
		return fmt.Sprintf("há %d min", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("há %d h", int(diff.Hours()))
	}
	if diff < 48*time.Hour {
		return "ontem"
	}
	return fmt.Sprintf("há %d dias", int(diff.Hours()/24))
}

func readKey() string {
	var buf [8]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil || n == 0 {
		return "QUIT"
	}

	if n == 1 {
		switch buf[0] {
		case 3:
			return "CTRL_C"
		case 27:
			return "ESC"
		case 9:
			return "TAB"
		case 10, 13:
			return "ENTER"
		default:
			return string(buf[0])
		}
	}

	if n >= 3 && buf[0] == 27 && buf[1] == '[' {
		switch buf[2] {
		case 'A':
			return "UP"
		case 'B':
			return "DOWN"
		case 'C':
			return "RIGHT"
		case 'D':
			return "LEFT"
		}
	}

	return "UNKNOWN"
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func showFallbackMenu() (*SelectionResult, error) {
	profiles, err := profile.List()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	fmt.Println("AI Sessions:")
	for i, p := range profiles {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		fmt.Printf("  %d) %-5s %-16s (%s - %s)\n", i+1, strings.ToUpper(p.Provider), p.Name, acc.Email, acc.Plan)
	}
	fmt.Print("Select profile (q to quit): ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "q" || text == "Q" || text == "" {
		return &SelectionResult{Action: ActionQuit}, nil
	}
	var n int
	if _, err := fmt.Sscanf(text, "%d", &n); err != nil || n < 1 || n > len(profiles) {
		return nil, fmt.Errorf("invalid selection")
	}
	p := profiles[n-1]
	return &SelectionResult{
		Action:      ActionRunProfile,
		Provider:    p.Provider,
		ProfileName: p.Name,
	}, nil
}

func setRaw(fd int) (*syscall.Termios, error) {
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	newState := oldState
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	return &oldState, nil
}

func restore(fd int, state *syscall.Termios) {
	if state != nil {
		_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(state)), 0, 0, 0)
	}
}
