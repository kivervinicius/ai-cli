package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAccountInfo(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	// Create AGY profile
	if _, err := Create("agy", "google-acc"); err != nil {
		t.Fatal(err)
	}
	homeAgy, _ := Home("agy", "google-acc")
	_ = os.MkdirAll(filepath.Join(homeAgy, ".gemini"), 0700)
	_ = os.WriteFile(filepath.Join(homeAgy, ".gemini", "google_accounts.json"), []byte(`{"active":"test@gmail.com"}`), 0600)

	infoAgy := GetAccountInfo("agy", "google-acc")
	if infoAgy.Email != "test@gmail.com" || !infoAgy.Authenticated {
		t.Fatalf("unexpected AGY account info: %+v", infoAgy)
	}

	// Create Codex profile
	if _, err := Create("codex", "openai-acc"); err != nil {
		t.Fatal(err)
	}
	homeCodex, _ := Home("codex", "openai-acc")
	// JWT with email claim
	jwt := "eyJhbGciOiJub25lIn0.eyJlbWFpbCI6ImNvZGV4QGV4YW1wbGUuY29tIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicHJvIn19.sig"
	_ = os.WriteFile(filepath.Join(homeCodex, "auth.json"), []byte(`{"auth_mode":"chatgpt","tokens":{"id_token":"`+jwt+`","access_token":"acc"}}`), 0600)

	infoCodex := GetAccountInfo("codex", "openai-acc")
	if infoCodex.Email != "codex@example.com" || infoCodex.Plan != "ChatGPT Pro" || !infoCodex.Authenticated {
		t.Fatalf("unexpected Codex account info: %+v", infoCodex)
	}
}
