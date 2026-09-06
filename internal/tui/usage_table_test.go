package tui

import "testing"

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
