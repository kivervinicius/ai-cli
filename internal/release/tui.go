package release

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type tuiModel struct {
	runner   Runner
	current  string
	branch   string
	commit   string
	options  []Option
	selected int
	custom   string
	step     string
	next     string
	result   Result
	err      error
}

type releaseDone struct {
	result Result
	err    error
}

func newTUI(r Runner, current, branch, commit string) tuiModel {
	return tuiModel{runner: r, current: current, branch: branch, commit: commit, options: Options(current), step: "select"}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step == "confirm" {
				m.step = "select"
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if m.step == "select" && m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.step == "select" && m.selected < len(m.options)-1 {
				m.selected++
			}
		case "backspace":
			if m.step == "custom" && len(m.custom) > 0 {
				m.custom = m.custom[:len(m.custom)-1]
			}
		case "enter":
			if m.step == "select" {
				if m.options[m.selected].Kind == "custom" {
					m.step = "custom"
				} else {
					m.next, m.err = NextVersion(m.current, m.options[m.selected].Kind)
					if m.err == nil {
						m.step = "confirm"
					}
				}
			} else if m.step == "custom" {
				m.next, m.err = m.custom, Validate(m.custom)
				if m.err == nil {
					m.step = "confirm"
				}
			} else if m.step == "confirm" {
				return m, m.execute()
			}
		default:
			if m.step == "custom" && len(msg.String()) == 1 && msg.Runes[0] >= 32 {
				m.custom += msg.String()
			}
		}
	case releaseDone:
		m.result, m.err = msg.result, msg.err
		if m.err != nil {
			m.step = "error"
		} else {
			m.step = "done"
		}
	}
	return m, nil
}

func (m tuiModel) execute() tea.Cmd {
	return func() tea.Msg {
		result, err := m.runner.Execute(context.Background(), m.current, m.next)
		return releaseDone{result, err}
	}
}

func (m tuiModel) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Nexus release\n\nCurrent version: %s\nBranch: %s\nCommit: %s\nFrontend: pending\nGo build: pending\n", m.current, m.branch, m.commit)
	switch m.step {
	case "select":
		b.WriteString("Choose release type (↑/↓ or j/k, Enter, Esc):\n\n")
		for i, option := range m.options {
			mark := "  "
			if i == m.selected {
				mark = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", mark, option.Label)
		}
	case "custom":
		fmt.Fprintf(&b, "Custom SemVer (Enter to continue, Esc to return):\n\n%s_\n", m.custom)
	case "confirm":
		fmt.Fprintf(&b, "\nNext version: %s\nArtifacts: VERSION, embedded frontend, Go binary, local nexus + ai\n\nPress Enter to build, Esc to return.\n", m.next)
	case "done":
		fmt.Fprintf(&b, "\nReleased %s\nFrontend: %s\nGo build: %s\nValidation: %s\nInstalled: %s\n", m.result.Version, m.result.Frontend, m.result.GoBuild, m.result.Validation, m.result.BinaryPath)
	case "error":
		fmt.Fprintf(&b, "\nRelease failed: %v\nVERSION was restored when possible.\n\nPress q or Esc to exit.\n", m.err)
	}
	return b.String()
}

func Run(root string) error {
	runner := NewRunner(root)
	current, err := runner.ReadVersion()
	if err != nil {
		return err
	}
	branch, commit := "unknown", "unknown"
	if out, gitErr := runner.Run(context.Background(), "git", "-C", root, "branch", "--show-current"); gitErr == nil {
		branch = strings.TrimSpace(string(out))
	}
	if out, gitErr := runner.Run(context.Background(), "git", "-C", root, "rev-parse", "--short", "HEAD"); gitErr == nil {
		commit = strings.TrimSpace(string(out))
	}
	p := tea.NewProgram(newTUI(runner, current, branch, commit))
	final, err := p.Run()
	if err != nil {
		return err
	}
	m := final.(tuiModel)
	if m.err != nil {
		return m.err
	}
	if m.step != "done" {
		return nil
	}
	fmt.Fprintln(os.Stdout, m.View())
	return nil
}
