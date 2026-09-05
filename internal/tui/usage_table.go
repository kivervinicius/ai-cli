package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// UsageTableRow is the presentation-only representation of a quota model group.
type UsageTableRow struct {
	Provider    string
	Profile     string
	Account     string
	Plan        string
	Group       string
	FiveHour    string
	Weekly      string
	Status      string
	ModelName   string
	LastUpdated string
}

// usageTableSelectMsg is sent when the user presses Enter on a row.
type usageTableSelectMsg struct {
	row *UsageTableRow
}

type usageTableModel struct {
	all      []UsageTableRow
	filtered []UsageTableRow
	table    table.Model
	filter   textinput.Model
	selected *UsageTableRow
	quitting bool
}

// RunUsageTable opens a searchable, keyboard-navigable quota table.
// Returns the selected row when the user presses Enter, or nil on quit.
func RunUsageTable(rows []UsageTableRow) (*UsageTableRow, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		printUsageTable(rows)
		return nil, nil
	}
	m, err := tea.NewProgram(newUsageTableModel(rows), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}
	fm := m.(usageTableModel)
	return fm.selected, nil
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

func newUsageTableModel(rows []UsageTableRow) usageTableModel {
	filter := textinput.New()
	filter.Placeholder = "filtrar por perfil, conta, provedor ou grupo"
	filter.Prompt = "/ "
	filter.CharLimit = 120
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)
	filtered := make([]UsageTableRow, len(rows))
	copy(filtered, rows)
	t := table.New(table.WithColumns(usageColumns(132)), table.WithRows(toUsageTableRows(filtered)), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(styles)
	return usageTableModel{all: rows, filtered: filtered, table: t, filter: filter}
}

func (m usageTableModel) Init() tea.Cmd { return nil }
func (m usageTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case usageTableSelectMsg:
		m.selected = msg.row
		m.quitting = true
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.table.SetColumns(usageColumns(msg.Width))
		m.table.SetWidth(max(56, msg.Width-4))
		m.table.SetHeight(max(5, msg.Height-6))
	case tea.KeyMsg:
		if m.filter.Focused() {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.filtered = filterUsageRows(m.all, m.filter.Value())
			m.table.SetRows(toUsageTableRows(m.filtered))
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(m.filtered) {
				row := m.filtered[idx]
				return m, func() tea.Msg { return usageTableSelectMsg{row: &row} }
			}
			return m, nil
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}
func (m usageTableModel) View() string {
	help := "↑/↓ navegar · enter selecionar · / filtrar · esc concluir filtro · q sair"
	if m.filter.Focused() {
		help = "Digite para filtrar · enter/esc concluir"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("Nexus · Uso por grupo de modelo")
	detail := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render("enter = abrir conta selecionada")
	return fmt.Sprintf("%s  %s\n%s\n\n%s\n%s", title, detail, m.filter.View(), m.table.View(), lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(help))
}
func usageColumns(width int) []table.Column {
	if width < 110 {
		return []table.Column{
			{Title: "PERFIL / CONTA", Width: 24},
			{Title: "MODELO", Width: 14},
			{Title: "5H", Width: 18},
			{Title: "SEMANA", Width: 18},
			{Title: "STATUS", Width: 12},
		}
	}
	return []table.Column{
		{Title: "PERFIL / CONTA", Width: 30},
		{Title: "MODELO", Width: 16},
		{Title: "5H (REST./RESET)", Width: 26},
		{Title: "SEMANAL (REST./RESET)", Width: 28},
		{Title: "STATUS", Width: 13},
	}
}
func toUsageTableRows(rows []UsageTableRow) []table.Row {
	result := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		identity := strings.Join([]string{row.Provider + ":" + row.Profile, row.Account, row.Plan}, " · ")
		model := row.ModelName
		if model == "" {
			model = "—"
		}
		if len(model) > 14 {
			model = model[:12] + ".."
		}
		result = append(result, table.Row{identity, model, row.FiveHour, row.Weekly, row.Status})
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
