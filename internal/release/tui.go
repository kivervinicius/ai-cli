package release

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type stepItem struct {
	title  string
	status string // "pending", "running", "done", "error"
}

type tuiModel struct {
	runner     Runner
	current    string
	branch     string
	commit     string
	options    []Option
	selected   int
	custom     string
	step       string // "select", "custom", "confirm", "building", "done", "error"
	next       string
	result     Result
	err        error
	spinnerIdx int
	startTime  time.Time
	steps      []stepItem
	progressCh chan progressMsg
}

type progressMsg struct {
	step   int
	total  int
	title  string
	status string
}

type tickMsg time.Time

type releaseDone struct {
	result Result
	err    error
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newTUI(r Runner, current, branch, commit string) tuiModel {
	return tuiModel{
		runner:  r,
		current: current,
		branch:  branch,
		commit:  commit,
		options: Options(current),
		step:    "select",
		steps: []stepItem{
			{title: "Compilando frontend web (Bun/Node)", status: "pending"},
			{title: "Compilando binário Go com LDFLAGS", status: "pending"},
			{title: fmt.Sprintf("Instalando em %s (nexus + ai)", r.LocalBin), status: "pending"},
			{title: "Validando versão do binário instalado", status: "pending"},
		},
		progressCh: make(chan progressMsg, 10),
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.step == "building" {
			m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
			return m, tickCmd()
		}
	case progressMsg:
		if msg.step >= 1 && msg.step <= len(m.steps) {
			m.steps[msg.step-1].title = msg.title
			m.steps[msg.step-1].status = msg.status
		}
		// Continue waiting for next progress message or completion
		return m, waitForProgress(m.progressCh)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step == "confirm" || m.step == "custom" {
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
			switch m.step {
			case "select":
				if m.options[m.selected].Kind == "custom" {
					m.step = "custom"
				} else {
					m.next, m.err = NextVersion(m.current, m.options[m.selected].Kind)
					if m.err == nil {
						m.step = "confirm"
					}
				}
			case "custom":
				m.next, m.err = m.custom, Validate(m.custom)
				if m.err == nil {
					m.step = "confirm"
				}
			case "confirm":
				m.step = "building"
				m.startTime = time.Now()
				return m, tea.Batch(m.execute(), tickCmd(), waitForProgress(m.progressCh))
			case "done", "error":
				return m, tea.Quit
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

func waitForProgress(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m tuiModel) execute() tea.Cmd {
	return func() tea.Msg {
		callback := func(step int, total int, title string, status string) {
			m.progressCh <- progressMsg{
				step:   step,
				total:  total,
				title:  title,
				status: status,
			}
		}
		result, err := m.runner.ExecuteWithProgress(context.Background(), m.current, m.next, callback)
		return releaseDone{result, err}
	}
}

func (m tuiModel) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== IAPro Nexus Release Manager ===\n\nVersão atual: %s | Branch: %s | Commit: %s\n\n", m.current, m.branch, m.commit)

	switch m.step {
	case "select":
		b.WriteString("Escolha o tipo de release (↑/↓ ou j/k, Enter para selecionar, Esc para sair):\n\n")
		for i, option := range m.options {
			mark := "  "
			if i == m.selected {
				mark = "❯ "
			}
			fmt.Fprintf(&b, "%s%s\n", mark, option.Label)
		}
	case "custom":
		fmt.Fprintf(&b, "Versão SemVer customizada (Enter para continuar, Esc para voltar):\n\n%s_\n", m.custom)
	case "confirm":
		fmt.Fprintf(&b, "Próxima versão: %s\nArtefatos: VERSION, frontend embutido, binário Go, link local ~/.local/bin/nexus + ai\n\n➔ Pressione [Enter] para iniciar a construção e instalação, ou [Esc] para voltar.\n", m.next)
	case "building":
		spinner := spinnerFrames[m.spinnerIdx]
		elapsed := time.Since(m.startTime).Round(100 * time.Millisecond)
		fmt.Fprintf(&b, "⏳ Construindo release %s... (%s)\nAguarde a finalização automática dos passos abaixo:\n\n", m.next, elapsed)
		for i, s := range m.steps {
			var icon string
			switch s.status {
			case "done":
				icon = "✓"
			case "running":
				icon = spinner
			case "error":
				icon = "✗"
			default:
				icon = "○"
			}
			fmt.Fprintf(&b, "  %s [%d/%d] %s\n", icon, i+1, len(m.steps), s.title)
		}
		b.WriteString("\n(Pressione Ctrl+C para abortar)\n")
	case "done":
		fmt.Fprintf(&b, "✓ Release v%s construído e instalado com sucesso!\n\n", m.result.Version)
		fmt.Fprintf(&b, "  • Frontend Web:      %s\n", m.result.Frontend)
		fmt.Fprintf(&b, "  • Binário Go:        %s\n", m.result.GoBuild)
		fmt.Fprintf(&b, "  • Validação:         %s\n", m.result.Validation)
		fmt.Fprintf(&b, "  • Binário Instalado: %s (alias: ai)\n\n", m.result.BinaryPath)
		b.WriteString("➔ Pressione [Enter], [q] ou [Esc] para fechar.\n")
	case "error":
		fmt.Fprintf(&b, "❌ Falha no release: %v\nO arquivo VERSION foi restaurado para o estado anterior (%s).\n\n", m.err, m.current)
		b.WriteString("➔ Pressione [q] ou [Esc] para fechar.\n")
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
