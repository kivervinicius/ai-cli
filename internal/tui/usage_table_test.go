package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterUsageRowsFindsAcrossIdentityAndGroup(t *testing.T) {
	rows := []UsageTableRow{{Provider: "agy", Profile: "work", Account: "work@example.com", Group: "Gemini Models"}, {Provider: "codex", Profile: "personal", Group: "GPT Models"}}
	if got := filterUsageRows(rows, "gemini"); len(got) != 1 || got[0].Profile != "work" {
		t.Fatalf("group filter = %#v", got)
	}
	if got := filterUsageRows(rows, "personal"); len(got) != 1 || got[0].Provider != "codex" {
		t.Fatalf("profile filter = %#v", got)
	}
}

func TestFilterUsageRowsIncludesUnknownProviders(t *testing.T) {
	rows := []UsageTableRow{
		{Provider: "agy", Profile: "work", FiveHour: "90%", Weekly: "80%", Status: "DISPONIVEL"},
		{Provider: "codex", Profile: "personal", FiveHour: "— desconhecido", Weekly: "— desconhecido", Status: "UNKNOWN"},
	}
	if got := filterUsageRows(rows, "codex"); len(got) != 1 || got[0].Status != "UNKNOWN" {
		t.Fatalf("expected codex UNKNOWN row, got %#v", got)
	}
	// Empty filter returns all
	if got := filterUsageRows(rows, ""); len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

func TestUnifiedUsageEscAndQQuit(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{{Provider: "codex", Profile: "work"}},
	}
	m := newUnifiedUsageModel(opts)

	// Test 'q' key
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit cmd on 'q'")
	}
	fm := res.(usageTableModel)
	if !fm.quitting || fm.chosenResult == nil || fm.chosenResult.Action != ActionQuit {
		t.Fatalf("expected ActionQuit on 'q', got %#v", fm.chosenResult)
	}

	// Test 'esc' key
	m2 := newUnifiedUsageModel(opts)
	res2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd2 == nil {
		t.Fatal("expected quit cmd on 'esc'")
	}
	fm2 := res2.(usageTableModel)
	if !fm2.quitting || fm2.chosenResult == nil || fm2.chosenResult.Action != ActionQuit {
		t.Fatalf("expected ActionQuit on 'esc', got %#v", fm2.chosenResult)
	}
}

func TestUnifiedUsageUnconfiguredCLIsRendered(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{
			{Provider: "codex", Profile: "work", Status: "DISPONIVEL"},
		},
		UnconfiguredCLIs: []InstalledProviderInfo{
			{ID: "claude", Name: "Claude Code", Version: "2.1.250", Installed: true, Profiles: 0},
			{ID: "cursor", Name: "Cursor CLI", Version: "2026.08", Installed: true, Profiles: 0},
		},
	}
	m := newUnifiedUsageModel(opts)
	if len(m.allAccounts) != 3 {
		t.Fatalf("expected 3 rows (1 configured + 2 unconfigured), got %d", len(m.allAccounts))
	}

	foundClaude := false
	for _, r := range m.allAccounts {
		if r.Provider == "claude" {
			foundClaude = true
			if !r.IsUnconfigured || r.Status != "NÃO CONFIGURADO" {
				t.Fatalf("expected Claude to be unconfigured, got %#v", r)
			}
		}
	}
	if !foundClaude {
		t.Fatal("expected unconfigured Claude row in table")
	}
}

func TestUnifiedUsageModeTogglingAndFlags(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows:        []UsageTableRow{{Provider: "codex", Profile: "work"}},
		InitialMode: ModeSafe,
	}
	m := newUnifiedUsageModel(opts)

	// Press '2' or 'y' for YOLO
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(usageTableModel)
	if m.execMode != ModeYOLO {
		t.Fatalf("expected YOLO mode, got %v", m.execMode)
	}

	// Press 'c' to toggle continue
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(usageTableModel)
	if !m.continueSession {
		t.Fatal("expected continueSession to be true")
	}

	// Press Enter to select
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected quit cmd on enter")
	}
	fm := res.(usageTableModel)
	if fm.chosenResult == nil {
		t.Fatal("expected non-nil chosenResult")
	}
	if fm.chosenResult.Action != ActionRunProfile {
		t.Fatalf("expected ActionRunProfile, got %v", fm.chosenResult.Action)
	}
	// Flags should include --yolo and --continue
	expectedFlags := []string{"--yolo", "--continue"}
	if len(fm.chosenResult.Args) != len(expectedFlags) {
		t.Fatalf("expected args %v, got %v", expectedFlags, fm.chosenResult.Args)
	}
	if fm.chosenResult.Args[0] != "--yolo" || fm.chosenResult.Args[1] != "--continue" {
		t.Fatalf("unexpected args: %v", fm.chosenResult.Args)
	}
}
