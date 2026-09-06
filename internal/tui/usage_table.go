package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/localization"
	"github.com/kivervinicius/ai-cli/internal/profile"
	"golang.org/x/term"
)

// ExecutionMode defines the operating mode for launched runtimes.
type ExecutionMode int

const (
	ModeSafe ExecutionMode = iota
	ModeYOLO
	ModePlan
)

func (m ExecutionMode) String() string {
	switch m {
	case ModeYOLO:
		return "YOLO"
	case ModePlan:
		return "Plan"
	default:
		return "Safe"
	}
}

// ActiveTab defines which view is displayed in the main table area.
type ActiveTab int

const (
	TabAccounts ActiveTab = iota
	TabSessions
)

// ModalMode defines the overlay modal currently open.
type ModalMode int

const (
	ModalNone ModalMode = iota
	ModalQuotaDetails
	ModalResumeChoice
)

// InstalledProviderInfo describes an AI CLI detected on the system.
type InstalledProviderInfo struct {
	ID        string
	Name      string
	Version   string
	Installed bool
	Profiles  int
}

// UsageTableRow is the presentation representation of an account quota window or provider row.
type UsageTableRow struct {
	Provider       string
	Profile        string
	Account        string
	Plan           string
	Group          string
	FiveHour       string
	Weekly         string
	Status         string
	ModelName      string
	LastUpdated    string
	IsUnconfigured bool
	IsDefault      bool
}

// UnifiedUsageOptions holds all context needed to run the unified TUI.
type UnifiedUsageOptions struct {
	Rows             []UsageTableRow
	UnconfiguredCLIs []InstalledProviderInfo
	Sessions         []conversation.Conversation
	Accounts         map[string]model.AccountInfo
	Defaults         map[string]string // provider -> default profile
	Workspace        string
	InitialMode      ExecutionMode
	InitialContinue  bool
}

// ActionType defines the user's intent upon exiting the TUI.
type ActionType int

const (
	ActionNone ActionType = iota
	ActionRunProfile
	ActionResumeConversation
	ActionSetDefault
	ActionLogin
	ActionQuit
)

// SelectionResult is returned when the user chooses an action.
type SelectionResult struct {
	Action         ActionType
	Provider       string
	ProfileName    string
	ConversationID string
	Args           []string
}

type usageTableModel struct {
	options UnifiedUsageOptions

	// State
	activeTab       ActiveTab
	execMode        ExecutionMode
	continueSession bool
	modalMode       ModalMode
	statusMsg       string

	// Accounts table
	allAccounts      []UsageTableRow
	filteredAccounts []UsageTableRow
	accountTable     table.Model

	// Sessions table
	allSessions      []conversation.Conversation
	filteredSessions []conversation.Conversation
	sessionTable     table.Model

	// Filter input
	filter textinput.Model

	// Selection & lifecycle
	selectedRow  *UsageTableRow
	selectedSess *conversation.Conversation
	chosenResult *SelectionResult
	quitting     bool
	width        int
	height       int
}

// RunUnifiedUsage launches the modern, unified Quotas & Runtimes TUI.
func RunUnifiedUsage(opts UnifiedUsageOptions) (*SelectionResult, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		printUsageTable(opts.Rows)
		return nil, nil
	}
	p := tea.NewProgram(newUnifiedUsageModel(opts), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm := m.(usageTableModel)
	if fm.chosenResult != nil {
		return fm.chosenResult, nil
	}
	return &SelectionResult{Action: ActionQuit}, nil
}

// RunUsageTable is a backward-compatible adapter that wraps RunUnifiedUsage.
func RunUsageTable(rows []UsageTableRow) (*UsageTableRow, error) {
	opts := UnifiedUsageOptions{
		Rows:        rows,
		InitialMode: ModeSafe,
	}
	res, err := RunUnifiedUsage(opts)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Action == ActionRunProfile && res.ProfileName != "" {
		for _, r := range rows {
			if r.Provider == res.Provider && r.Profile == res.ProfileName {
				return &r, nil
			}
		}
		return &UsageTableRow{Provider: res.Provider, Profile: res.ProfileName}, nil
	}
	return nil, nil
}

func printUsageTable(rows []UsageTableRow) {
	fmt.Println("Nexus · Uso por grupo de modelo")
	fmt.Printf("%-10s %-20s %-20s %-14s %-20s %-20s %-12s\n", "PROVEDOR", "PERFIL", "CONTA", "PLANO", "5H", "SEMANA", "STATUS")
	for _, row := range rows {
		account := row.Account
		if len(account) > 18 {
			account = account[:16] + ".."
		}
		plan := row.Plan
		if len(plan) > 12 {
			plan = plan[:10] + ".."
		}
		fmt.Printf("%-10s %-20s %-20s %-14s %-20s %-20s %-12s\n", row.Provider, row.Profile, account, plan, row.FiveHour, row.Weekly, row.Status)
	}
}

func newUnifiedUsageModel(opts UnifiedUsageOptions) usageTableModel {
	// Combine configured account rows with installed but unconfigured CLIs
	allRows := make([]UsageTableRow, 0, len(opts.Rows)+len(opts.UnconfiguredCLIs))
	allRows = append(allRows, opts.Rows...)

	// Mark default accounts
	for i := range allRows {
		if opts.Defaults[allRows[i].Provider] == allRows[i].Profile {
			allRows[i].IsDefault = true
		}
	}

	// Add unconfigured installed CLIs as informative rows
	for _, cli := range opts.UnconfiguredCLIs {
		if cli.Installed && cli.Profiles == 0 {
			vStr := cli.Version
			if vStr == "" {
				vStr = "instalado"
			}
			allRows = append(allRows, UsageTableRow{
				Provider:       cli.ID,
				Profile:        "(sem perfil)",
				Account:        cli.Name + " · " + vStr,
				Plan:           "—",
				Group:          "—",
				FiveHour:       "—",
				Weekly:         "—",
				Status:         "NÃO CONFIGURADO",
				ModelName:      "—",
				IsUnconfigured: true,
			})
		}
	}

	filter := textinput.New()
	filter.Placeholder = "filtrar por perfil, conta, provedor ou modelo (/)"
	filter.Prompt = "/ "
	filter.CharLimit = 100

	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)

	// Accounts table setup
	filteredAccounts := make([]UsageTableRow, len(allRows))
	copy(filteredAccounts, allRows)
	accTable := table.New(table.WithColumns(usageColumns(132)), table.WithRows(toUsageTableRows(filteredAccounts)), table.WithFocused(true), table.WithHeight(10))
	accTable.SetStyles(styles)

	// Sessions table setup
	allSess := opts.Sessions
	filteredSess := make([]conversation.Conversation, len(allSess))
	copy(filteredSess, allSess)
	sessTable := table.New(table.WithColumns(sessionColumns(132)), table.WithRows(toSessionTableRows(filteredSess)), table.WithFocused(false), table.WithHeight(10))
	sessTable.SetStyles(styles)

	return usageTableModel{
		options:          opts,
		activeTab:        TabAccounts,
		execMode:         opts.InitialMode,
		continueSession:  opts.InitialContinue,
		modalMode:        ModalNone,
		allAccounts:      allRows,
		filteredAccounts: filteredAccounts,
		accountTable:     accTable,
		allSessions:      allSess,
		filteredSessions: filteredSess,
		sessionTable:     sessTable,
		filter:           filter,
		width:            120,
		height:           30,
	}
}

func (m usageTableModel) Init() tea.Cmd { return nil }

func (m usageTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.accountTable.SetColumns(usageColumns(msg.Width))
		m.accountTable.SetWidth(max(56, msg.Width-4))
		m.accountTable.SetHeight(max(6, msg.Height-16))

		m.sessionTable.SetColumns(sessionColumns(msg.Width))
		m.sessionTable.SetWidth(max(56, msg.Width-4))
		m.sessionTable.SetHeight(max(6, msg.Height-16))

	case tea.KeyMsg:
		k := msg.String()

		// Global quit: ctrl+c always exits immediately
		if k == "ctrl+c" {
			m.quitting = true
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Handle active modal
		if m.modalMode != ModalNone {
			if k == "esc" || k == "q" || k == "Q" || k == "enter" || k == " " {
				m.modalMode = ModalNone
				return m, nil
			}
			return m, nil
		}

		// Handle search/filter focus
		if m.filter.Focused() {
			if k == "esc" {
				m.filter.Blur()
				return m, nil
			}
			if k == "enter" {
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			q := m.filter.Value()
			if m.activeTab == TabAccounts {
				m.filteredAccounts = filterUsageRows(m.allAccounts, q)
				m.accountTable.SetRows(toUsageTableRows(m.filteredAccounts))
			} else {
				m.filteredSessions = filterSessionRows(m.allSessions, q)
				m.sessionTable.SetRows(toSessionTableRows(m.filteredSessions))
			}
			return m, cmd
		}

		// Main screen keys: esc or q exits immediately!
		if k == "esc" || k == "q" || k == "Q" {
			m.quitting = true
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Tab switching between Accounts and Sessions
		if k == "tab" || k == "shift+tab" {
			if m.activeTab == TabAccounts {
				m.activeTab = TabSessions
				m.accountTable.Blur()
				m.sessionTable.Focus()
			} else {
				m.activeTab = TabAccounts
				m.sessionTable.Blur()
				m.accountTable.Focus()
			}
			return m, nil
		}

		// Mode switching keys
		switch k {
		case "1":
			m.execMode = ModeSafe
			m.statusMsg = "Modo alterado para Safe (padrão interativo supervisionado)"
			return m, nil
		case "2", "y", "Y":
			m.execMode = ModeYOLO
			m.statusMsg = "Modo alterado para ⚡ YOLO (auto-aprovação e bypass de sandbox)"
			return m, nil
		case "3", "p", "P":
			m.execMode = ModePlan
			m.statusMsg = "Modo alterado para 📋 Plan (modo planejamento)"
			return m, nil
		case "m", "M":
			m.execMode = (m.execMode + 1) % 3
			m.statusMsg = fmt.Sprintf("Modo alterado para %s", m.execMode.String())
			return m, nil
		case "c", "C":
			m.continueSession = !m.continueSession
			if m.continueSession {
				m.statusMsg = "Opção Continuar Sessão: LIGADA (--continue)"
			} else {
				m.statusMsg = "Opção Continuar Sessão: DESLIGADA"
			}
			return m, nil
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		case "s", "S":
			// Open Quota Details modal for highlighted account
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				m.modalMode = ModalQuotaDetails
			}
			return m, nil
		case "d", "D":
			// Set as default profile
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if !row.IsUnconfigured {
						_ = config.SetDefaultProfile(row.Provider, row.Profile)
						for i := range m.allAccounts {
							if m.allAccounts[i].Provider == row.Provider {
								m.allAccounts[i].IsDefault = (m.allAccounts[i].Profile == row.Profile)
							}
						}
						for i := range m.filteredAccounts {
							if m.filteredAccounts[i].Provider == row.Provider {
								m.filteredAccounts[i].IsDefault = (m.filteredAccounts[i].Profile == row.Profile)
							}
						}
						m.accountTable.SetRows(toUsageTableRows(m.filteredAccounts))
						m.statusMsg = fmt.Sprintf("✓ Definido %s:%s como perfil padrão.", row.Provider, row.Profile)
					}
				}
			}
			return m, nil
		case "l", "L":
			// Trigger login
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if !row.IsUnconfigured {
						m.quitting = true
						m.chosenResult = &SelectionResult{
							Action:      ActionLogin,
							Provider:    row.Provider,
							ProfileName: row.Profile,
						}
						return m, tea.Quit
					}
				}
			}
			return m, nil
		case "a", "A":
			// Hint on adding profile
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					m.statusMsg = fmt.Sprintf("Para configurar %s: execute 'nexus add %s <nome>'", row.Provider, row.Provider)
				}
			}
			return m, nil
		case "enter":
			if m.activeTab == TabAccounts {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if row.IsUnconfigured {
						m.statusMsg = fmt.Sprintf("⚠ %s não está configurado. Execute: nexus add %s <nome>", row.Provider, row.Provider)
						return m, nil
					}
					var flags []string
					if m.execMode == ModeYOLO {
						flags = append(flags, "--yolo")
					} else if m.execMode == ModePlan {
						flags = append(flags, "--plan")
					}
					if m.continueSession {
						flags = append(flags, "--continue")
					}
					m.quitting = true
					m.chosenResult = &SelectionResult{
						Action:      ActionRunProfile,
						Provider:    row.Provider,
						ProfileName: row.Profile,
						Args:        flags,
					}
					return m, tea.Quit
				}
			} else if m.activeTab == TabSessions {
				idx := m.sessionTable.Cursor()
				if idx >= 0 && idx < len(m.filteredSessions) {
					sess := m.filteredSessions[idx]
					var flags []string
					if m.execMode == ModeYOLO {
						flags = append(flags, "--yolo")
					} else if m.execMode == ModePlan {
						flags = append(flags, "--plan")
					}
					m.quitting = true
					m.chosenResult = &SelectionResult{
						Action:         ActionResumeConversation,
						Provider:       sess.Provider,
						ProfileName:    "",
						ConversationID: sess.ID,
						Args:           flags,
					}
					return m, tea.Quit
				}
			}
			return m, nil
		}
	}

	// Update table models
	var cmd tea.Cmd
	if m.activeTab == TabAccounts {
		m.accountTable, cmd = m.accountTable.Update(msg)
	} else {
		m.sessionTable, cmd = m.sessionTable.Update(msg)
	}
	return m, cmd
}

func (m usageTableModel) View() string {
	if m.quitting {
		return ""
	}

	availWidth := m.width
	if availWidth < 80 {
		availWidth = 80
	}

	// Handle Modals
	if m.modalMode == ModalQuotaDetails {
		return m.renderQuotaModal(availWidth)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	planStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))

	// Header line
	cwd := m.options.Workspace
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	wsBasename := filepath.Base(cwd)
	headerLeft := fmt.Sprintf("%s %s", titleStyle.Render("Nexus · Workspace OS"), subStyle.Render("v0.4.0"))
	headerRight := subStyle.Render("workspace: ~/" + wsBasename)
	gap := availWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2
	if gap < 2 {
		gap = 2
	}
	header := headerLeft + strings.Repeat(" ", gap) + headerRight

	// Mode Toolbar (OPÇÕES NA TELA)
	safePill := "[○ 1: Safe]"
	yoloPill := "[○ 2: ⚡ YOLO]"
	planPill := "[○ 3: 📋 Plan]"

	switch m.execMode {
	case ModeSafe:
		safePill = accentStyle.Render("[● 1: Safe (Padrão)]")
	case ModeYOLO:
		yoloPill = warnStyle.Render("[● 2: ⚡ YOLO (Bypass & Auto-approve)]")
	case ModePlan:
		planPill = planStyle.Render("[● 3: 📋 Plan (Planejamento)]")
	}

	contPill := "[c] Continuar Sessão: OFF"
	if m.continueSession {
		contPill = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("[c] Continuar Sessão: ON (--continue)")
	}

	modeToolbar := fmt.Sprintf("  MODO:  %s  %s  %s    OPÇÕES:  %s", safePill, yoloPill, planPill, contPill)

	// Mode Explanation Banner
	modeDesc := "🛡️  Modo Safe: Execução interativa supervisionada com pedidos de confirmação."
	if m.execMode == ModeYOLO {
		modeDesc = warnStyle.Render("⚡ Modo YOLO: Auto-aprova permissões e bypassa sandbox (--yolo).")
	} else if m.execMode == ModePlan {
		modeDesc = planStyle.Render("📋 Modo Plan: Inicia o runtime em modo de análise e planejamento (--plan).")
	}
	modeDescLine := "  " + modeDesc

	// Tabs Header
	tab1Label := fmt.Sprintf("CONTAS & QUOTAS (%d)", len(m.filteredAccounts))
	tab2Label := fmt.Sprintf("SESSÕES RECENTES (%d)", len(m.filteredSessions))
	tab1 := subStyle.Render("[ 1: " + tab1Label + " ]")
	tab2 := subStyle.Render("[ 2: " + tab2Label + " ]")

	if m.activeTab == TabAccounts {
		tab1 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Render(" [● 1: " + tab1Label + "] ")
	} else {
		tab2 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Render(" [● 2: " + tab2Label + "] ")
	}
	tabsLine := fmt.Sprintf("  %s  %s   %s", tab1, tab2, subStyle.Render("(pressione Tab para alternar)"))

	// Active table view
	var tableContent string
	if m.activeTab == TabAccounts {
		tableContent = m.accountTable.View()
	} else {
		tableContent = m.sessionTable.View()
	}

	// Dynamic Selection Preview
	var previewLine string
	if m.activeTab == TabAccounts {
		idx := m.accountTable.Cursor()
		if idx >= 0 && idx < len(m.filteredAccounts) {
			row := m.filteredAccounts[idx]
			if row.IsUnconfigured {
				previewLine = fmt.Sprintf("  Provedor %s instalado mas sem perfil. Pressione [a] ou execute: nexus add %s <nome>", row.Provider, row.Provider)
			} else {
				flagNote := "nenhuma flag extra"
				if m.execMode == ModeYOLO && m.continueSession {
					flagNote = "flags: --yolo --continue"
				} else if m.execMode == ModeYOLO {
					flagNote = "flags: --yolo"
				} else if m.execMode == ModePlan && m.continueSession {
					flagNote = "flags: --plan --continue"
				} else if m.execMode == ModePlan {
					flagNote = "flags: --plan"
				} else if m.continueSession {
					flagNote = "flags: --continue"
				}
				previewLine = fmt.Sprintf("  ▶ [Enter] Disparar %s:%s no modo %s (%s)", row.Provider, row.Profile, m.execMode.String(), flagNote)
			}
		}
	} else {
		idx := m.sessionTable.Cursor()
		if idx >= 0 && idx < len(m.filteredSessions) {
			sess := m.filteredSessions[idx]
			previewLine = fmt.Sprintf("  ▶ [Enter] Retomar sessão %s (%s) no modo %s", sess.ID[:min(8, len(sess.ID))], sess.Provider, m.execMode.String())
		}
	}

	// Status line if any
	status := ""
	if m.statusMsg != "" {
		status = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render(m.statusMsg) + "\n"
	}

	// Bottom action keys bar
	help := subStyle.Render("  [↑/↓] Navegar  [Tab] Alternar Aba  [1-3/y/p] Alternar Modo  [c] Continuar  [s] Detalhes  [d] Padrão  [l] Login  [/] Filtrar  [Esc/q] Sair")
	if m.filter.Focused() {
		help = subStyle.Render("  Digite para filtrar  [Enter/Esc] Concluir filtro")
	}

	return fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n\n%s\n\n%s\n\n%s%s\n",
		header,
		modeToolbar,
		modeDescLine,
		tabsLine,
		tableContent,
		previewLine,
		status,
		help,
	)
}

func (m usageTableModel) renderQuotaModal(width int) string {
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

	idx := m.accountTable.Cursor()
	if idx < 0 || idx >= len(m.filteredAccounts) {
		return modalStyle.Render("Nenhuma conta selecionada.\n\n[Pressione Esc ou q para voltar]")
	}
	row := m.filteredAccounts[idx]
	if row.IsUnconfigured {
		return modalStyle.Render(fmt.Sprintf("Provedor %s ainda não possui perfis.\nExecute: nexus add %s <nome>\n\n[Pressione Esc ou q para voltar]", row.Provider, row.Provider))
	}

	acc := m.options.Accounts[row.Provider+":"+row.Profile]
	qv := profile.GetQuotaView(row.Provider, row.Profile, acc.Plan, acc.Email)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("📊 Limites & Quota — %s:%s", row.Provider, row.Profile)) + "\n\n")
	fmt.Fprintf(&sb, " Conta:   %s (%s)\n", accentStyle.Render(acc.Email), magentaStyle.Render(acc.Plan))
	fmt.Fprintf(&sb, " Status:  %s\n", acc.Status)

	availLabel := qv.AvailabilityLabel()
	availColor := "42"
	if !qv.IsAvailable() {
		availColor = "208"
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
	sb.WriteString(subStyle.Render("Pressione qualquer tecla ou Esc/q para voltar"))

	return modalStyle.Render(sb.String())
}

func usageColumns(width int) []table.Column {
	if width < 110 {
		return []table.Column{
			{Title: "PROVEDOR", Width: 10},
			{Title: "PERFIL / CONTA", Width: 24},
			{Title: "MODELO", Width: 14},
			{Title: "5H", Width: 18},
			{Title: "SEMANA", Width: 18},
			{Title: "STATUS", Width: 14},
		}
	}
	return []table.Column{
		{Title: "PROVEDOR", Width: 11},
		{Title: "PERFIL / CONTA", Width: 30},
		{Title: "MODELO", Width: 16},
		{Title: "5H (REST./RESET)", Width: 26},
		{Title: "SEMANAL (REST./RESET)", Width: 28},
		{Title: "STATUS", Width: 16},
	}
}

func sessionColumns(width int) []table.Column {
	if width < 110 {
		return []table.Column{
			{Title: "PROVEDOR", Width: 10},
			{Title: "SESSÃO", Width: 10},
			{Title: "MODIFICADO", Width: 12},
			{Title: "TÍTULO", Width: 26},
			{Title: "WORKSPACE", Width: 16},
		}
	}
	return []table.Column{
		{Title: "PROVEDOR", Width: 11},
		{Title: "SESSÃO", Width: 14},
		{Title: "MODIFICADO", Width: 16},
		{Title: "TÍTULO", Width: 48},
		{Title: "WORKSPACE", Width: 26},
	}
}

func toUsageTableRows(rows []UsageTableRow) []table.Row {
	result := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		provBadge := formatProviderBadge(row.Provider)
		star := ""
		if row.IsDefault {
			star = " ★"
		}

		identity := row.Profile + star
		if row.Account != "" && !row.IsUnconfigured {
			identity += " · " + row.Account
		} else if row.IsUnconfigured {
			identity = row.Account
		}

		model := row.ModelName
		if model == "" {
			model = "—"
		}
		if len(model) > 15 {
			model = model[:13] + ".."
		}

		result = append(result, table.Row{provBadge, identity, model, row.FiveHour, row.Weekly, row.Status})
	}
	return result
}

func toSessionTableRows(sess []conversation.Conversation) []table.Row {
	result := make([]table.Row, 0, len(sess))
	for _, s := range sess {
		provBadge := formatProviderBadge(s.Provider)
		sID := s.ID
		if len(sID) > 10 {
			sID = sID[:8] + ".."
		}
		timeStr := formatTimeAgo(s.LastModified)
		title := s.Title
		if len(title) > 46 {
			title = title[:44] + ".."
		}
		wsShort := filepath.Base(s.Workspace)
		if wsShort == "" || wsShort == "." {
			wsShort = "~"
		}
		result = append(result, table.Row{provBadge, sID, timeStr, title, "~/" + wsShort})
	}
	return result
}

func filterUsageRows(rows []UsageTableRow, query string) []UsageTableRow {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}
	filtered := make([]UsageTableRow, 0, len(rows))
	for _, row := range rows {
		search := strings.ToLower(strings.Join([]string{
			row.Provider, row.Profile, row.Account, row.Plan,
			row.Group, row.FiveHour, row.Weekly, row.Status, row.ModelName,
		}, " "))
		if strings.Contains(search, query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterSessionRows(sess []conversation.Conversation, query string) []conversation.Conversation {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return sess
	}
	var out []conversation.Conversation
	for _, s := range sess {
		search := strings.ToLower(strings.Join([]string{
			s.Provider, s.ID, s.Title, s.Workspace,
		}, " "))
		if strings.Contains(search, query) {
			out = append(out, s)
		}
	}
	return out
}

func formatProviderBadge(p string) string {
	switch strings.ToLower(p) {
	case "codex":
		return "[CODEX ]"
	case "agy":
		return "[AGY   ]"
	case "claude":
		return "[CLAUDE]"
	case "opencode":
		return "[OPENCD]"
	case "gemini":
		return "[GEMINI]"
	case "cursor":
		return "[CURSOR]"
	default:
		s := strings.ToUpper(p)
		if len(s) > 6 {
			s = s[:6]
		}
		return fmt.Sprintf("[%-6s]", s)
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
