package agy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkSharedAgyItemsDoesNotOverwriteProfileSettings(t *testing.T) {
	host := t.TempDir()
	profile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(host, "antigravity-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profile, ".gemini", "antigravity-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"settings.json": `{"model":"new-model"}`,
		"history.jsonl": "history\n",
	} {
		if err := os.WriteFile(filepath.Join(host, "antigravity-cli", name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	linkSharedAgyItems(profile, host)

	if _, err := os.Stat(filepath.Join(profile, ".gemini", "antigravity-cli", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("profile settings must remain profile-owned, stat error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(profile, ".gemini", "antigravity-cli", "history.jsonl")); err != nil {
		t.Fatalf("history should still be shared: %v", err)
	}
}
