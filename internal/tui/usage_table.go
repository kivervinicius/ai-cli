package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UsageTableRow is the presentation-only representation of a quota model group.
type UsageTableRow struct{ Provider, Profile, Account, Plan, Group, FiveHour, Weekly, Status string }

type usageTableModel struct {
	all    []UsageTableRow
	table  table.Model
	filter textinput.Model
}

// RunUsageTable opens a searchable, keyboard-navigable quota table.
func RunUsageTable(rows []UsageTableRow) error {
	_, err := tea.NewProgram(newUsageTableModel(rows), tea.WithAltScreen()).Run()
	return err
}

func newUsageTableModel(rows []UsageTableRow) usageTableModel {
	filter := textinput.New()
	filter.Placeholder = "filtrar por perfil, conta, provedor ou grupo"
	filter.Prompt = "/ "
	filter.CharLimit = 120
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)
	t := table.New(table.WithColumns(usageColumns(132)), table.WithRows(toUsageTableRows(rows)), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(styles)
	return usageTableModel{all: rows, table: t, filter: filter}
}

func (m usageTableModel) Init() tea.Cmd { return nil }
func (m usageTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			m.table.SetRows(toUsageTableRows(filterUsageRows(m.all, m.filter.Value())))
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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
	help := "↑/↓ navegar · / filtrar · esc concluir filtro · q sair"
	if m.filter.Focused() {
		help = "Digite para filtrar · enter/esc concluir"
	}
	return fmt.Sprintf("%s\n%s\n\n%s\n%s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("Nexus · Uso por grupo de modelo"), m.filter.View(), m.table.View(), lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(help))
}
func usageColumns(width int) []table.Column {
	if width < 96 {
		return []table.Column{{Title: "PERFIL", Width: 20}, {Title: "GRUPO", Width: 15}, {Title: "5H", Width: 20}, {Title: "SEMANA", Width: 20}, {Title: "STATUS", Width: 12}}
	}
	return []table.Column{{Title: "PERFIL / CONTA", Width: 29}, {Title: "GRUPO", Width: 22}, {Title: "5H (REST./RESET)", Width: 25}, {Title: "SEMANAL (REST./RESET)", Width: 28}, {Title: "STATUS", Width: 13}}
}
func toUsageTableRows(rows []UsageTableRow) []table.Row {
	result := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		identity := strings.Join([]string{row.Provider + ":" + row.Profile, row.Account, row.Plan}, " · ")
		result = append(result, table.Row{identity, row.Group, row.FiveHour, row.Weekly, row.Status})
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
		if strings.Contains(strings.ToLower(strings.Join([]string{row.Provider, row.Profile, row.Account, row.Plan, row.Group, row.FiveHour, row.Weekly, row.Status}, " ")), query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
