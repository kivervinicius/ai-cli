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
