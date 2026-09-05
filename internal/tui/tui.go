package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/localization"
	"github.com/kivervinicius/ai-cli/internal/profile"
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

type Panel int

const (
	PanelProviders Panel = 0
	PanelAccounts  Panel = 1
	PanelSessions  Panel = 2
)

type ModalMode int

const (
	ModalNone ModalMode = iota
	ModalQuotaDetails
	ModalResumeChoice
)

type Model struct {
	width        int
	height       int
	activePanel  Panel
	modalMode    ModalMode
	selProvIndex int
	selAccIndex  int
	selSessIndex int
	modalSelIdx  int

	isSearching bool
	filterQuery string
	statusMsg   string

	providers []string
	profiles  []model.Profile
	accounts  map[string]model.AccountInfo
	sessions  []conversation.Conversation
	cfg       config.Config
	selector  *scheduler.Selector
	quotaEng  *quota.Engine
	cooldown  *cooldown.Tracker

	chosenResult *SelectionResult
}

func InitialModel() Model {
	cfg, _ := config.LoadConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cd := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cd)

	provs := []string{"codex", "agy", "claude", "opencode", "gemini", "cursor"}
	profs, _ := profile.List()

	accs := make(map[string]model.AccountInfo)
	for _, p := range profs {
		accs[p.Provider+":"+p.Name] = profile.GetAccountInfo(p.Provider, p.Name)
	}

	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(20, cwd)

	return Model{
		width:        100,
		height:       32,
		activePanel:  PanelAccounts,
		modalMode:    ModalNone,
		selProvIndex: 0,
		selAccIndex:  0,
		selSessIndex: 0,
		modalSelIdx:  0,
		providers:    provs,
		profiles:     profs,
		accounts:     accs,
		sessions:     convs,
		cfg:          cfg,
		selector:     sel,
		quotaEng:     qEng,
		cooldown:     cd,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) currentProvider() string {
	if len(m.providers) == 0 {
		return "codex"
	}
	return m.providers[m.selProvIndex%len(m.providers)]
}

func (m Model) filteredProfiles() []model.Profile {
	curProv := m.currentProvider()
	var out []model.Profile
	for _, p := range m.profiles {
		if p.Provider == curProv {
			if m.filterQuery == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(m.filterQuery)) {
				out = append(out, p)
			}
		}
	}
	return out
}

func (m Model) filteredSessions() []conversation.Conversation {
	if m.filterQuery == "" {
		return m.sessions
	}
	var out []conversation.Conversation
	q := strings.ToLower(m.filterQuery)
	for _, s := range m.sessions {
		if strings.Contains(strings.ToLower(s.Title), q) ||
			strings.Contains(strings.ToLower(s.Provider), q) ||
			strings.Contains(strings.ToLower(s.Workspace), q) ||
			strings.Contains(strings.ToLower(s.ID), q) {
			out = append(out, s)
		}
	}
	return out
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress && m.modalMode == ModalNone {
			if msg.Y >= 2 && msg.Y <= 11 {
				if msg.X < 26 {
					m.activePanel = PanelProviders
				} else {
					m.activePanel = PanelAccounts
				}
			} else if msg.Y > 11 && msg.Y <= 20 {
				m.activePanel = PanelSessions
			}
		}
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		// Global quit
		if k == "ctrl+c" {
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Search mode input
		if m.isSearching {
			switch k {
			case "esc", "enter":
				m.isSearching = false
			case "backspace":
				if len(m.filterQuery) > 0 {
					m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
				}
			default:
				if len(k) == 1 {
					m.filterQuery += k
				}
			}
			return m, nil
		}

		// Modal handling
		if m.modalMode != ModalNone {
			if k == "esc" || k == "q" || k == "Q" {
				m.modalMode = ModalNone
				return m, nil
			}

			if m.modalMode == ModalQuotaDetails {
				if k == "enter" || k == " " {
					m.modalMode = ModalNone
					return m, nil
				}
			}

			if m.modalMode == ModalResumeChoice {
				switch k {
				case "up", "k":
					if len(m.profiles) > 0 {
						m.modalSelIdx = (m.modalSelIdx - 1 + len(m.profiles)) % len(m.profiles)
					}
				case "down", "j":
					if len(m.profiles) > 0 {
						m.modalSelIdx = (m.modalSelIdx + 1) % len(m.profiles)
					}
				case "enter":
					sess := m.filteredSessions()
					if len(sess) > 0 && len(m.profiles) > 0 {
						s := sess[m.selSessIndex%len(sess)]
						chosenProf := m.profiles[m.modalSelIdx%len(m.profiles)]
						m.chosenResult = &SelectionResult{
							Action:         ActionResumeConversation,
							Provider:       chosenProf.Provider,
							ProfileName:    chosenProf.Name,
							ConversationID: s.ID,
						}
						return m, tea.Quit
					}
				case "1", "2", "3", "4", "5", "6", "7", "8", "9":
					idx := int(k[0] - '1')
					sess := m.filteredSessions()
					if len(sess) > 0 && idx < len(m.profiles) {
						s := sess[m.selSessIndex%len(sess)]
						chosenProf := m.profiles[idx]
						m.chosenResult = &SelectionResult{
							Action:         ActionResumeConversation,
							Provider:       chosenProf.Provider,
							ProfileName:    chosenProf.Name,
							ConversationID: s.ID,
						}
						return m, tea.Quit
					}
				}
				return m, nil
			}
		}

		// Main screen quit
		if k == "q" || k == "Q" || k == "esc" {
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Quick numeric execution (1-9)
		if k >= "1" && k <= "9" {
			idx := int(k[0] - '1')
			switch m.activePanel {
			case PanelAccounts:
				profs := m.filteredProfiles()
				if idx < len(profs) {
					p := profs[idx]
					// Check availability before launching.
					snap, _ := m.quotaEng.GetCachedUsage(p.Provider, p.Name)
					acc := m.accounts[p.Provider+":"+p.Name]
					qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
					if !qv.IsAvailable() {
						reason := qv.AvailabilityLabel()
						m.statusMsg = fmt.Sprintf("⚠ %s:%s indisponível (%s). Selecione outra conta.", p.Provider, p.Name, reason)
						return m, nil
					}
					m.chosenResult = &SelectionResult{
						Action:      ActionRunProfile,
						Provider:    p.Provider,
						ProfileName: p.Name,
					}
					return m, tea.Quit
				}
			case PanelSessions:
				sess := m.filteredSessions()
				if idx < len(sess) {
					m.selSessIndex = idx
					m.modalMode = ModalResumeChoice
					m.modalSelIdx = 0
					return m, nil
				}
			}
		}

		switch k {
		case "tab":
			m.activePanel = (m.activePanel + 1) % 3

		case "shift+tab":
			m.activePanel = (m.activePanel + 2) % 3

		case "left", "h":
			if m.activePanel == PanelAccounts {
				m.activePanel = PanelProviders
			}

		case "right":
			if m.activePanel == PanelProviders {
				m.activePanel = PanelAccounts
			}

		case "up", "k":
			switch m.activePanel {
			case PanelProviders:
				if len(m.providers) > 0 {
					m.selProvIndex = (m.selProvIndex - 1 + len(m.providers)) % len(m.providers)
					m.selAccIndex = 0
				}
			case PanelAccounts:
				profs := m.filteredProfiles()
				if len(profs) > 0 {
					m.selAccIndex = (m.selAccIndex - 1 + len(profs)) % len(profs)
				}
			case PanelSessions:
				sess := m.filteredSessions()
				if len(sess) > 0 {
					m.selSessIndex = (m.selSessIndex - 1 + len(sess)) % len(sess)
				}
			}

		case "down", "j":
			switch m.activePanel {
			case PanelProviders:
				if len(m.providers) > 0 {
					m.selProvIndex = (m.selProvIndex + 1) % len(m.providers)
					m.selAccIndex = 0
				}
			case PanelAccounts:
				profs := m.filteredProfiles()
				if len(profs) > 0 {
					m.selAccIndex = (m.selAccIndex + 1) % len(profs)
				}
			case PanelSessions:
				sess := m.filteredSessions()
				if len(sess) > 0 {
					m.selSessIndex = (m.selSessIndex + 1) % len(sess)
				}
			}

		case "enter", "o":
			if m.activePanel == PanelSessions {
				sess := m.filteredSessions()
				if len(sess) > 0 {
					m.modalMode = ModalResumeChoice
					m.modalSelIdx = 0
					return m, nil
				}
			} else {
				profs := m.filteredProfiles()
				if len(profs) > 0 {
					p := profs[m.selAccIndex%len(profs)]
					// Check availability before launching.
					snap, _ := m.quotaEng.GetCachedUsage(p.Provider, p.Name)
					acc := m.accounts[p.Provider+":"+p.Name]
					qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
					if !qv.IsAvailable() {
						reason := qv.AvailabilityLabel()
						m.statusMsg = fmt.Sprintf("⚠ %s:%s indisponível (%s). Selecione outra conta.", p.Provider, p.Name, reason)
						return m, nil
					}
					m.chosenResult = &SelectionResult{
						Action:      ActionRunProfile,
						Provider:    p.Provider,
						ProfileName: p.Name,
					}
					return m, tea.Quit
				} else {
					m.chosenResult = &SelectionResult{
						Action:      ActionRunProfile,
						Provider:    m.currentProvider(),
						ProfileName: "",
					}
					return m, tea.Quit
				}
			}

		case "c", "C":
			if len(m.sessions) > 0 {
				conv := m.sessions[0]
				profs := m.filteredProfiles()
				profName := ""
				provName := conv.Provider
				if len(profs) > 0 {
					profName = profs[m.selAccIndex%len(profs)].Name
					provName = profs[m.selAccIndex%len(profs)].Provider
				}
				m.chosenResult = &SelectionResult{
					Action:         ActionResumeConversation,
					Provider:       provName,
					ProfileName:    profName,
					ConversationID: conv.ID,
				}
				return m, tea.Quit
			}

		case "r", "R":
			sess := m.filteredSessions()
			if len(sess) > 0 {
				m.modalMode = ModalResumeChoice
				m.modalSelIdx = 0
			}

		case "s", "S":
			profs := m.filteredProfiles()
			if len(profs) > 0 {
				m.modalMode = ModalQuotaDetails
			}

		case "d", "D":
			profs := m.filteredProfiles()
			if len(profs) > 0 {
				p := profs[m.selAccIndex%len(profs)]
				_ = config.SetDefaultProfile(p.Provider, p.Name)
				m.cfg, _ = config.LoadConfig()
				m.statusMsg = fmt.Sprintf("✓ Definido %s:%s como perfil padrão.", p.Provider, p.Name)
			}

		case "l", "L":
			profs := m.filteredProfiles()
			if len(profs) > 0 {
				p := profs[m.selAccIndex%len(profs)]
				m.chosenResult = &SelectionResult{
					Action:      ActionLogin,
					Provider:    p.Provider,
					ProfileName: p.Name,
				}
				return m, tea.Quit
			}

		case "/":
			m.isSearching = true
			m.filterQuery = ""
		}
	}

	return m, nil
}

func (m Model) View() string {
	availWidth := m.width
	if availWidth < 88 {
		availWidth = 88
	}
	if availWidth > 120 {
		availWidth = 120
	}

	activeBorder := lipgloss.Color("39")    // Cyan/Blue
	inactiveBorder := lipgloss.Color("240") // Dark gray

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	selTextStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activeDot := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("●")
	inactiveDot := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○")
	warnStar := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")).Render("★")
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	btnStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")).Padding(0, 1)

	// Render Modals if active
	if m.modalMode == ModalQuotaDetails {
		return m.renderQuotaModal(availWidth)
	}
	if m.modalMode == ModalResumeChoice {
		return m.renderResumeModal(availWidth)
	}

	provWidth := 24
	accWidth := availWidth - provWidth - 3
	if accWidth < 58 {
		accWidth = 58
	}

	provBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inactiveBorder).
		Width(provWidth).
		Height(8).
		Padding(0, 1)

	if m.activePanel == PanelProviders {
		provBoxStyle = provBoxStyle.BorderForeground(activeBorder)
	}

	accBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inactiveBorder).
		Width(accWidth).
		Height(8).
		Padding(0, 1)

	if m.activePanel == PanelAccounts {
		accBoxStyle = accBoxStyle.BorderForeground(activeBorder)
	}

	sessBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inactiveBorder).
		Width(availWidth-2).
		Height(8).
		Padding(0, 1)

	if m.activePanel == PanelSessions {
		sessBoxStyle = sessBoxStyle.BorderForeground(activeBorder)
	}

	cwd, _ := os.Getwd()
	projName := filepath.Base(cwd)

	counts := make(map[string]int)
	for _, p := range m.profiles {
		counts[p.Provider]++
	}

	var sb strings.Builder

	// Header line
	headerLeft := fmt.Sprintf("%s %s", titleStyle.Render(localization.T("tui.title")), subStyle.Render("v0.4.0"))
	headerRight := subStyle.Render(localization.T("tui.workspace", map[string]any{"Name": "~/" + projName}))
	gap := availWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2
	if gap < 2 {
		gap = 2
	}
	sb.WriteString(headerLeft + strings.Repeat(" ", gap) + headerRight + "\n\n")

	// 1. Providers List
	var provLines []string
	provLines = append(provLines, lipgloss.NewStyle().Bold(true).Render(localization.T("tui.providers")))
	for i, pr := range m.providers {
		c := counts[pr]
		dot := inactiveDot
		if c > 0 {
			dot = activeDot
		}
		pName := formatProviderName(pr)
		pointer := "  "
		if i == m.selProvIndex {
			pointer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("▸ ")
		}

		nameCol := fmt.Sprintf("%-10s", pName)
		if i == m.selProvIndex {
			nameCol = selTextStyle.Render(nameCol)
		}
		countCol := subStyle.Render(fmt.Sprintf("%2d", c))

		provLines = append(provLines, fmt.Sprintf("%s%s %s %s", pointer, dot, nameCol, countCol))
	}
	provBox := provBoxStyle.Render(strings.Join(provLines, "\n"))

	// 2. Accounts List
	var accLines []string
	accTitle := localization.T("tui.accounts", map[string]any{"Provider": formatProviderName(m.currentProvider())})
	accLines = append(accLines, lipgloss.NewStyle().Bold(true).Render(accTitle))
	profs := m.filteredProfiles()
	if len(profs) == 0 {
		accLines = append(accLines,
			subStyle.Render("  Nenhum perfil cadastrado."),
			subStyle.Render("  Execute: ai add "+m.currentProvider()+" <nome>"))
	} else {
		for i, p := range profs {
			if i >= 5 {
				break
			}
			acc := m.accounts[p.Provider+":"+p.Name]
			isDef := m.cfg.Defaults[p.Provider] == p.Name

			pointer := "  "
			numBadge := subStyle.Render(fmt.Sprintf("[%d]", i+1))
			if i == m.selAccIndex {
				pointer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("▸ ")
			}

			dot := activeDot
			if !acc.Authenticated {
				dot = inactiveDot
			}

			nameStr := p.Name
			if len(nameStr) > 17 {
				nameStr = nameStr[:15] + ".."
			}
			nameCol := fmt.Sprintf("%-17s", nameStr)
			if i == m.selAccIndex {
				nameCol = selTextStyle.Render(nameCol)
			}

			planStr := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render(fmt.Sprintf("%-12s", acc.Plan))

			qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
			bar := ""
			if summary := qv.CompactGroupSummary(); summary != "" {
				bar = summary
			} else {
				bottleneck, _ := qv.Bottleneck()
				bar = quota.RenderBarWithPercent(bottleneck, qv.Status, 10)
			}
			if bar == "" {
				bar = "[ UNKNOWN  ] UNK"
			}
			availTag := ""
			if !qv.IsAvailable() {
				availTag = warnStyle.Render(" " + qv.AvailabilityLabel())
			}
			barStr := subStyle.Render(bar) + availTag

			starStr := " "
			if isDef {
				starStr = warnStar
			}

			accLines = append(accLines, fmt.Sprintf("%s%s %s %s %s %s %s", pointer, dot, numBadge, nameCol, planStr, barStr, starStr))
		}
	}
	accBox := accBoxStyle.Render(strings.Join(accLines, "\n"))

	// Side-by-side Top Section with 1 space gap
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, provBox, " ", accBox) + "\n\n")

	// 3. Sessions Box
	var sessLines []string
	sessLines = append(sessLines, lipgloss.NewStyle().Bold(true).Render(localization.T("tui.sessions")))
	filteredSess := m.filteredSessions()
	if len(filteredSess) == 0 {
		sessLines = append(sessLines, subStyle.Render("  Nenhuma sessão recente encontrada."))
	} else {
		for i, s := range filteredSess {
			if i >= 5 {
				break
			}
			timeStr := subStyle.Render(fmt.Sprintf("%-10s", formatTimeAgo(s.LastModified)))
			title := s.Title
			if len(title) > 34 {
				title = title[:32] + ".."
			}
			titleCol := fmt.Sprintf("%-34s", title)

			pointer := "  "
			numBadge := subStyle.Render(fmt.Sprintf("[%d]", i+1))
			if i == m.selSessIndex {
				pointer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("▸ ")
				titleCol = selTextStyle.Render(titleCol)
			}

			provBadge := formatProviderBadge(s.Provider)

			wsShort := filepath.Base(s.Workspace)
			if wsShort == "" || wsShort == "." {
				wsShort = "~"
			}
			wsCol := subStyle.Render("~/" + wsShort)

			sessLines = append(sessLines, fmt.Sprintf("%s%s %s  %s  %s  %s", pointer, numBadge, timeStr, titleCol, provBadge, wsCol))
		}
	}
	sessBox := sessBoxStyle.Render(strings.Join(sessLines, "\n"))
	sb.WriteString(sessBox + "\n\n")

	// 4. Selected Info & Status
	selProv := m.currentProvider()
	selProf := "(auto-select)"
	authBadge := subStyle.Render("[Unauthenticated]")
	if len(profs) > 0 {
		p := profs[m.selAccIndex%len(profs)]
		selProf = p.Name
		acc := m.accounts[p.Provider+":"+p.Name]
		if acc.Authenticated {
			authBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("[Authenticated]")
		}
	}
	fmt.Fprintf(&sb, " Selected: %s / %s  %s\n\n",
		titleStyle.Render(formatProviderName(selProv)),
		selTextStyle.Render(selProf),
		authBadge)

	// Action buttons row
	fmt.Fprintf(&sb, " %s %s %s %s %s %s\n\n",
		btnStyle.Render("[Enter/1-9] Run"),
		btnStyle.Render("[c] Continue Latest"),
		btnStyle.Render("[r] Resume Modal"),
		btnStyle.Render("[s] Quotas"),
		btnStyle.Render("[d] Default"),
		btnStyle.Render("[l] Login"))

	// 5. Smart Account Selection Banner
	autoText := fmt.Sprintf("%s %s is optimal for new sessions (highest available capacity)",
		activeDot, selProf)
	sb.WriteString(" " + autoText + "\n\n")

	// 6. Footer Controls / Search
	if m.statusMsg != "" {
		sb.WriteString(" " + activeDot + " " + m.statusMsg + "\n\n")
	}

	if m.isSearching {
		sb.WriteString(" Busca rápida: " + m.filterQuery + "█\n")
	} else {
		controls := subStyle.Render(" [↑/↓] Navegar  [←/→/Tab] Trocar Caixa  [1-9] Disparo Rápido  [/] Buscar  [q/Esc] Sair")
		sb.WriteString(" " + controls)
	}

	return sb.String()
}

func (m Model) renderQuotaModal(width int) string {
	boxWidth := width - 4
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Width(boxWidth).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	magentaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	profs := m.filteredProfiles()
	if len(profs) == 0 {
		return modalStyle.Render("Nenhum perfil selecionado.\n\n[Pressione Esc ou q para voltar]")
	}
	p := profs[m.selAccIndex%len(profs)]
	acc := m.accounts[p.Provider+":"+p.Name]
	qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("📊 Limites & Quota — %s (%s)", p.Name, strings.ToUpper(p.Provider))) + "\n\n")
	fmt.Fprintf(&sb, " Conta:   %s (%s)\n", accentStyle.Render(acc.Email), magentaStyle.Render(acc.Plan))
	fmt.Fprintf(&sb, " Status:  %s\n", acc.Status)

	// Availability line
	availLabel := qv.AvailabilityLabel()
	availColor := "42" // green
	if !qv.IsAvailable() {
		availColor = "208" // orange
	}
	availStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(availColor))
	fmt.Fprintf(&sb, " Quota:   %s\n\n", availStyle.Render(availLabel))

	sb.WriteString(accentStyle.Render("MODELOS & CAPACIDADE:") + "\n")
	for _, group := range qv.ModelGroups {
		if qv.HasMultipleGroups() && group.Name != "" {
			fmt.Fprintf(&sb, "\n  %s\n", groupStyle.Render(group.Name+":"))
		}
		for _, w := range group.Windows {
			fmt.Fprintf(&sb, "  %s: %s\n", w.Label, w.Bar)
			if w.ResetDesc != "" {
				sb.WriteString(subStyle.Render(fmt.Sprintf("                  Reset em %s\n", w.ResetDesc)))
			}
		}
	}
	sb.WriteString("\n")

	sb.WriteString(subStyle.Render("Capacidades: Usage: ✓  Resume: ✓  Isolated Runtime: ✓  Project Binding: ✓\n\n"))
	sb.WriteString(titleStyle.Render("[ Pressione qualquer tecla ou Esc para voltar ]"))

	return modalStyle.Render(sb.String())
}

func (m Model) renderResumeModal(width int) string {
	boxWidth := width - 4
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("213")).
		Width(boxWidth).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	sess := m.filteredSessions()
	if len(sess) == 0 {
		return modalStyle.Render("Nenhuma sessão selecionada.\n\n[Pressione Esc para voltar]")
	}
	chosenSess := sess[m.selSessIndex%len(sess)]

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Continuar \"%s\" (%s) com qual conta?", chosenSess.Title, chosenSess.ID[:8])) + "\n\n")

	// Suggestion from smart selector
	cwd, _ := os.Getwd()
	suggestedResult, _ := m.selector.SelectBestProfile(context.Background(), chosenSess.Provider, cwd, m.profiles, m.accounts, nil)
	if suggestedResult != nil && suggestedResult.SelectedProfile != nil {
		fmt.Fprintf(&sb, " 💡 %s Sugerido: %s (%s)\n\n",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("AUTO:"),
			selStyle.Render(suggestedResult.SelectedProfile.Name),
			suggestedResult.Reason)
	}

	sb.WriteString(subStyle.Render("Selecione a conta para retomar a sessão:") + "\n\n")
	for i, p := range m.profiles {
		if i >= 8 {
			break
		}
		pointer := "  "
		if i == m.modalSelIdx {
			pointer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("▸ ")
		}
		numBadge := subStyle.Render(fmt.Sprintf("[%d]", i+1))
		provBadge := formatProviderBadge(p.Provider)
		acc := m.accounts[p.Provider+":"+p.Name]
		nameCol := fmt.Sprintf("%-18s", p.Name)
		if i == m.modalSelIdx {
			nameCol = selStyle.Render(nameCol)
		}
		emailCol := subStyle.Render(fmt.Sprintf("%-26s", acc.Email))
		planCol := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render(fmt.Sprintf("%-12s", acc.Plan))

		fmt.Fprintf(&sb, "%s%s %s %s %s %s\n", pointer, numBadge, provBadge, nameCol, emailCol, planCol)
	}

	sb.WriteString("\n" + titleStyle.Render("[Enter/1-9]") + " Iniciar nesta conta    " + subStyle.Render("[Esc/q] Cancelar"))

	return modalStyle.Render(sb.String())
}

func formatProviderBadge(p string) string {
	switch strings.ToLower(p) {
	case "codex":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("[CODEX ]")
	case "agy":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("[AGY   ]")
	case "claude":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")).Render("[CLAUDE]")
	case "opencode":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33")).Render("[OPENCD]")
	case "gemini":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("[GEMINI]")
	default:
		return fmt.Sprintf("[%s]", strings.ToUpper(p))
	}
}

func formatProviderName(p string) string {
	switch strings.ToLower(p) {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	case "agy":
		return "AGY"
	case "opencode":
		return "OpenCode"
	case "gemini":
		return "Gemini"
	default:
		return p
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return localization.T("tui.unknown")
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return localization.T("tui.now")
	}
	if diff < time.Hour {
		return localization.T("tui.minutes_ago", map[string]any{"Count": int(diff.Minutes())})
	}
	if diff < 24*time.Hour {
		return localization.T("tui.hours_ago", map[string]any{"Count": int(diff.Hours())})
	}
	if diff < 48*time.Hour {
		return localization.T("tui.yesterday")
	}
	return localization.T("tui.days_ago", map[string]any{"Count": int(diff.Hours() / 24)})
}

// ShowMenu launches the interactive control plane TUI.
func ShowMenu() (*SelectionResult, error) {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	if m, ok := finalModel.(Model); ok {
		if m.chosenResult != nil {
			return m.chosenResult, nil
		}
	}
	return &SelectionResult{Action: ActionQuit}, nil
}
