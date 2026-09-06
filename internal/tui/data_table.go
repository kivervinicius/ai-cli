package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DataTableColumn and DataTableRow keep terminal table presentation generic and
// independent from domain packages.
type DataTableColumn struct {
	Title string
	Width int
}
type DataTableRow struct {
	Values     []string
	SearchText string
}
type DataTableOptions struct {
	Title, FilterPlaceholder string
	Columns                  []DataTableColumn
	Rows                     []DataTableRow
}

// RunDataTable renders the shared searchable table used by human CLI output.
func RunDataTable(options DataTableOptions) error {
	_, err := tea.NewProgram(newDataTableModel(options), tea.WithAltScreen()).Run()
	return err
}

type dataTableModel struct {
	options DataTableOptions
	table   table.Model
	filter  textinput.Model
}

func newDataTableModel(options DataTableOptions) dataTableModel {
	filter := textinput.New()
	filter.Placeholder = options.FilterPlaceholder
	filter.Prompt = "/ "
	filter.CharLimit = 120
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)
	t := table.New(table.WithColumns(toTableColumns(options.Columns)), table.WithRows(toTableRows(options.Rows)), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(styles)
	return dataTableModel{options: options, table: t, filter: filter}
}
func (m dataTableModel) Init() tea.Cmd { return nil }
func (m dataTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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
			m.table.SetRows(toTableRows(filterDataTableRows(m.options.Rows, m.filter.Value())))
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
func (m dataTableModel) View() string {
	help := "↑/↓ navegar · / filtrar · esc concluir filtro · q sair"
	if m.filter.Focused() {
		help = "Digite para filtrar · enter/esc concluir"
	}
	return fmt.Sprintf("%s\n%s\n\n%s\n%s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render(m.options.Title), m.filter.View(), m.table.View(), lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(help))
}
func toTableColumns(columns []DataTableColumn) []table.Column {
	out := make([]table.Column, 0, len(columns))
	for _, column := range columns {
		out = append(out, table.Column{Title: column.Title, Width: column.Width})
	}
	return out
}
func toTableRows(rows []DataTableRow) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		out = append(out, table.Row(row.Values))
	}
	return out
}
func filterDataTableRows(rows []DataTableRow, query string) []DataTableRow {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}
	out := make([]DataTableRow, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.SearchText), query) {
			out = append(out, row)
		}
	}
	return out
}
