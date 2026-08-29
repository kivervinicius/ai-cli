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
	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/localization"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Bold(true)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4B5563")).
			Padding(0, 1)

	activeBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#8B5CF6")).
				Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#374151"))

	statusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	statusWaitingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	statusStoppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	statusFailedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
)

type Tab int

const (
	TabRuntimes Tab = iota
	TabEvents
)

// ControlModel is the Bubble Tea model for the AI Control Center TUI.
type ControlModel struct {
	width         int
	height        int
	activeTab     Tab
	runtimes      []registry.RuntimeSession
	selectedIndex int
	eventsList    []events.Event
	workspace     string
	statusMessage string
	statusTime    time.Time
	quitting      bool
	attachTarget  string
}

// NewControlModel creates an initial ControlModel.
func NewControlModel() ControlModel {
	cwd, _ := os.Getwd()
	reg := registry.DefaultRegistry()
	_, _ = reg.CleanupStale()
	runtimes := reg.List()

	evts := events.DefaultBus().GetHistory("*", 50)

	return ControlModel{
		activeTab:     TabRuntimes,
		runtimes:      runtimes,
		selectedIndex: 0,
		eventsList:    evts,
		workspace:     cwd,
	}
}

func (m ControlModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func (m ControlModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		reg := registry.DefaultRegistry()
		m.runtimes = reg.List()
		m.eventsList = events.DefaultBus().GetHistory("*", 50)
		if m.selectedIndex >= len(m.runtimes) && len(m.runtimes) > 0 {
			m.selectedIndex = len(m.runtimes) - 1
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "tab":
			if m.activeTab == TabRuntimes {
				m.activeTab = TabEvents
			} else {
				m.activeTab = TabRuntimes
			}
			return m, nil

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.runtimes)-1 {
				m.selectedIndex++
			}
			return m, nil

		case "r":
			reg := registry.DefaultRegistry()
			_, _ = reg.CleanupStale()
			m.runtimes = reg.List()
			m.statusMessage = localization.T("tui.refreshed")
			m.statusTime = time.Now()
			return m, nil

		case "s":
			if len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
				target := m.runtimes[m.selectedIndex]
				reg := registry.DefaultRegistry()
				client, err := protocol.NewClient(target.RuntimeID)
				if err == nil {
					_ = client.Stop()
					_ = client.Close()
					m.statusMessage = fmt.Sprintf("✓ Sent stop signal to %s", target.RuntimeID)
				} else {
					// Fallback: kill PID if running and clean up record
					if target.PID > 0 && registry.IsProcessAlive(target.PID) {
						if p, pErr := os.FindProcess(target.PID); pErr == nil {
							_ = p.Kill()
						}
					}
					_ = os.Remove(protocol.EndpointPath(target.RuntimeID))
					_ = reg.Delete(target.RuntimeID)
					m.runtimes = reg.List()
					if m.selectedIndex >= len(m.runtimes) && len(m.runtimes) > 0 {
						m.selectedIndex = len(m.runtimes) - 1
					}
					m.statusMessage = fmt.Sprintf("✓ Cleaned up dead runtime %s", target.RuntimeID)
				}
				m.statusTime = time.Now()
			}
			return m, nil

		case "d", "x", "delete", "backspace":
			if len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
				target := m.runtimes[m.selectedIndex]
				reg := registry.DefaultRegistry()
				if target.PID > 0 && registry.IsProcessAlive(target.PID) {
					if p, pErr := os.FindProcess(target.PID); pErr == nil {
						_ = p.Kill()
					}
				}
				_ = os.Remove(protocol.EndpointPath(target.RuntimeID))
				_ = reg.Delete(target.RuntimeID)
				m.runtimes = reg.List()
				if m.selectedIndex >= len(m.runtimes) && len(m.runtimes) > 0 {
					m.selectedIndex = len(m.runtimes) - 1
				}
				m.statusMessage = fmt.Sprintf("✓ Deleted runtime %s", target.RuntimeID)
				m.statusTime = time.Now()
			}
			return m, nil

		case "c":
			reg := registry.DefaultRegistry()
			_, _ = reg.CleanupStale()
			purged, _ := reg.PurgeInactive()
			m.runtimes = reg.List()
			m.selectedIndex = 0
			m.statusMessage = fmt.Sprintf("✓ Cleaned up %d stale runtime records", purged)
			m.statusTime = time.Now()
			return m, nil

		case "a", "enter":
			if len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
				m.attachTarget = m.runtimes[m.selectedIndex].RuntimeID
				m.quitting = true
				return m, tea.Quit
			}
		case "h":
			if len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
				m.statusMessage = localization.T("tui.handoff")
				m.statusTime = time.Now()
			}
			return m, nil
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if msg.Y >= 4 && msg.Y < 4+len(m.runtimes) {
				m.selectedIndex = msg.Y - 4
			}
		}
	}

	return m, nil
}

func (m ControlModel) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	// 1. Top Title Bar
	wsBasename := filepath.Base(m.workspace)
	headerLeft := titleStyle.Render("⚡ AI CONTROL CENTER")
	headerRight := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(localization.T("tui.workspace", map[string]any{"Name": wsBasename}))
	gap := w - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 4
	if gap < 2 {
		gap = 2
	}
	topBar := fmt.Sprintf(" %s%s%s\n", headerLeft, strings.Repeat(" ", gap), headerRight)

	// 2. Tabs
	tab1 := "[ 1. Runtimes ]"
	tab2 := "[ 2. Events & Logs ]"
	if m.activeTab == TabRuntimes {
		tab1 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")).Render(tab1)
		tab2 = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(tab2)
	} else {
		tab1 = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(tab1)
		tab2 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")).Render(tab2)
	}
	tabsRow := fmt.Sprintf("  %s   %s\n\n", tab1, tab2)

	var content string
	if m.activeTab == TabRuntimes {
		content = m.renderRuntimesTab(w)
	} else {
		content = m.renderEventsTab(w)
	}

	// 3. Status and shortcuts bar
	statusText := m.statusMessage
	if time.Since(m.statusTime) > 3*time.Second {
		statusText = ""
	}
	if statusText != "" {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("ℹ " + statusText + "\n")
	}

	shortcutsStr := " [a/Enter] Attach"

	if m.activeTab == TabRuntimes && len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
		sel := m.runtimes[m.selectedIndex]
		if d, err := driver.DefaultRegistry().Get(sel.ProviderID); err == nil {
			caps := d.EffectiveCaps(context.Background(), model.Profile{Name: sel.ProfileID, Provider: sel.ProviderID})
			if caps.Process.Status == driver.CapabilitySupported || caps.Terminal.Status == driver.CapabilitySupported {
				shortcutsStr += "   [s] Stop"
			}
			if caps.Resume.Status == driver.CapabilitySupported {
				shortcutsStr += "   [h] Handoff"
			}
		} else {
			shortcutsStr += "   [s] Stop"
		}
	} else {
		shortcutsStr += "   [s] Stop" // Default for empty/events tab
	}

	shortcutsStr += "   [d] Delete   [c] Clean Stale   [r] Refresh   [Tab] Switch Tab   [q] Quit"

	shortcuts := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(shortcutsStr)

	return topBar + tabsRow + content + "\n" + statusText + shortcuts + "\n"
}

func (m ControlModel) renderRuntimesTab(width int) string {
	var sb strings.Builder

	header := fmt.Sprintf("  %-16s %-10s %-14s %-12s %-12s %s\n", "RUNTIME ID", "PROVIDER", "PROFILE", "STATE", "CONTROL", "PID")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("  " + strings.Repeat("─", max(20, width-6)) + "\n")

	if len(m.runtimes) == 0 {
		sb.WriteString("\n   (No managed runtimes running. Start one with: nexus start <provider>)\n\n")
		return sb.String()
	}

	for i, s := range m.runtimes {
		stateStr := string(s.State)
		switch s.State {
		case registry.StateRunning:
			stateStr = statusRunningStyle.Render(stateStr)
		case registry.StateWaiting, registry.StateApproval:
			stateStr = statusWaitingStyle.Render(stateStr)
		case registry.StateStopped, registry.StateStale:
			stateStr = statusStoppedStyle.Render(stateStr)
		default:
			stateStr = statusFailedStyle.Render(stateStr)
		}

		line := fmt.Sprintf("  %-16s %-10s %-14s %-12s %-12s %d",
			truncate(s.RuntimeID, 15),
			truncate(strings.ToUpper(s.ProviderID), 9),
			truncate(s.ProfileID, 13),
			stateStr,
			truncate(string(s.ControlLevel), 11),
			s.PID,
		)

		if i == m.selectedIndex {
			sb.WriteString(selectedRowStyle.Render("▸ "+line[2:]) + "\n")
		} else {
			sb.WriteString("  " + line[2:] + "\n")
		}
	}

	// Details box for selected runtime
	if len(m.runtimes) > 0 && m.selectedIndex < len(m.runtimes) {
		sel := m.runtimes[m.selectedIndex]
		sb.WriteString("\n")
		details := fmt.Sprintf(
			"Runtime: %s | Provider: %s:%s | State: %s | Level: %s | Host PID: %d | Child PID: %d\nWorkspace: %s\nEndpoint:  %s",
			sel.RuntimeID, strings.ToUpper(sel.ProviderID), sel.ProfileID, sel.State, sel.ControlLevel, sel.HostPID, sel.PID,
			sel.Workspace, sel.ControlEndpoint,
		)
		sb.WriteString(activeBorderStyle.Render(details) + "\n")
	}

	return sb.String()
}

func (m ControlModel) renderEventsTab(width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("  RECENT EVENT LOG (Real-Time)\n"))
	sb.WriteString("  " + strings.Repeat("─", max(20, width-6)) + "\n")

	if len(m.eventsList) == 0 {
		sb.WriteString("\n   (No recorded events yet)\n\n")
		return sb.String()
	}

	for _, e := range m.eventsList {
		timeStr := e.Timestamp.Format("15:04:05")
		line := fmt.Sprintf("  [%s] %-14s %-18s %s", timeStr, e.Type, fmt.Sprintf("%s:%s", e.Provider, e.Profile), e.Summary)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

// RunControlTUI launches the interactive AI Control Center TUI.
func RunControlTUI(ctx context.Context) (attachTargetID string, err error) {
	p := tea.NewProgram(NewControlModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}
	if cm, ok := finalModel.(ControlModel); ok {
		return cm.attachTarget, nil
	}
	return "", nil
}
