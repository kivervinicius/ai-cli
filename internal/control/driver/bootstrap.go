package driver

import (
	"os"
	"path/filepath"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
)

// bootstrapProfile prepares a profile with non-credential conversation
// artifacts so a new supervised runtime can see prior local context.
func bootstrapProfile(provider string, p model.Profile) (string, error) {
	home, err := config.ProfileHome(provider, p.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return "", err
	}
	cfgObj, _ := config.LoadConfig()
	if err := security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset)); err != nil {
		return "", err
	}
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return home, nil
	}
	switch provider {
	case "codex":
		linkConversationArtifacts(home, filepath.Join(hostHome, ".codex"), []string{
			"session_index.jsonl", "thread_history_1.sqlite", "thread_history_1.sqlite-wal", "thread_history_1.sqlite-shm", "sessions",
		})
	case "agy":
		linkConversationArtifacts(filepath.Join(home, ".gemini"), filepath.Join(hostHome, ".gemini"), []string{
			"antigravity-cli/history.jsonl", "antigravity-cli/conversation_summaries.db",
		})
	case "opencode":
		linkConversationArtifacts(filepath.Join(home, ".local", "share", "opencode"), filepath.Join(hostHome, ".local", "share", "opencode"), []string{
			"session", "sessions", "storage/session", "storage/message",
		})
	}
	return home, nil
}

func linkConversationArtifacts(destinationRoot, sourceRoot string, items []string) {
	for _, item := range items {
		src := filepath.Join(sourceRoot, item)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := security.SafeLinkOrCopy(src, filepath.Join(destinationRoot, item)); err != nil {
			// Best-effort continuity; missing history must not block bootstrap.
			continue
		}
	}
}
