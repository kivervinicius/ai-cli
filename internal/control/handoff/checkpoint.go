package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/security"
)

// WorkCheckpoint encapsulates a structured, bounded, and redacted snapshot of active work.
type WorkCheckpoint struct {
	SchemaVersion   int       `json:"schema_version"`
	CheckpointID    string    `json:"checkpoint_id"`
	SourceRuntimeID string    `json:"source_runtime_id"`
	SourceProvider  string    `json:"source_provider"`
	SourceProfile   string    `json:"source_profile"`
	SourceSessionID string    `json:"source_session_id,omitempty"`
	SourceModel     string    `json:"source_model,omitempty"`
	Workspace       string    `json:"workspace"`
	Goal            string    `json:"goal,omitempty"`
	GitBranch       string    `json:"git_branch,omitempty"`
	GitStatus       string    `json:"git_status,omitempty"`
	GitDiffStat     string    `json:"git_diff_stat,omitempty"`
	ChangedFiles    []string  `json:"changed_files,omitempty"`
	Decisions       []string  `json:"decisions,omitempty"`
	PendingTasks    []string  `json:"pending_tasks,omitempty"`
	Tests           []string  `json:"tests,omitempty"`
	Errors          []string  `json:"errors,omitempty"`
	Commands        []string  `json:"commands,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// LineageRecord tracks handoff relationships across runtimes and session IDs.
type LineageRecord struct {
	LineageID               string    `json:"lineage_id"`
	SourceRuntimeID         string    `json:"source_runtime_id"`
	SourceProviderSessionID string    `json:"source_provider_session_id,omitempty"`
	TargetRuntimeID         string    `json:"target_runtime_id"`
	TargetProviderSessionID string    `json:"target_provider_session_id,omitempty"`
	Type                    string    `json:"type"` // "ACCOUNT_HANDOFF" | "CONTEXT_HANDOFF"
	CreatedAt               time.Time `json:"created_at"`
	CheckpointID            string    `json:"checkpoint_id,omitempty"`
}

// CaptureWorkCheckpoint captures bounded workspace metadata with universal redaction.
func CaptureWorkCheckpoint(workspace, sourceRuntimeID, sourceProvider, sourceProfile, sourceSessionID, sourceModel, goal string) WorkCheckpoint {
	id := fmt.Sprintf("cp-%s-%d", sourceProvider, time.Now().UnixNano())
	cp := WorkCheckpoint{
		SchemaVersion:   3,
		CheckpointID:    id,
		SourceRuntimeID: sourceRuntimeID,
		SourceProvider:  sourceProvider,
		SourceProfile:   sourceProfile,
		SourceSessionID: sourceSessionID,
		SourceModel:     sourceModel,
		Workspace:       security.Redact(workspace),
		Goal:            security.Redact(goal),
		CreatedAt:       time.Now(),
	}

	// 1. Bounded git branch
	if out, err := exec.Command("git", "-C", workspace, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		cp.GitBranch = security.Redact(strings.TrimSpace(string(out)))
	}

	// 2. Bounded git status (up to 50 files)
	if out, err := exec.Command("git", "-C", workspace, "status", "--short").Output(); err == nil {
		rawStatus := string(out)
		cp.GitStatus = security.Redact(rawStatus)

		lines := strings.Split(rawStatus, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 3 {
				cp.ChangedFiles = append(cp.ChangedFiles, security.Redact(strings.TrimSpace(line[3:])))
				if len(cp.ChangedFiles) >= 50 {
					break
				}
			}
		}
	}

	// 3. Bounded git diff stat (max 20 KB)
	if out, err := exec.Command("git", "-C", workspace, "diff", "--stat").Output(); err == nil {
		diffStat := string(out)
		if len(diffStat) > 20*1024 {
			diffStat = diffStat[:20*1024] + "\n... (truncated diff stat)"
		}
		cp.GitDiffStat = security.Redact(diffStat)
	}

	return cp
}

// SaveCheckpoint writes the checkpoint to persistent storage.
func SaveCheckpoint(cp WorkCheckpoint) (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), "ai-control")
	}
	cpDir := filepath.Join(dataDir, "checkpoints")
	_ = os.MkdirAll(cpDir, 0700)

	file := filepath.Join(cpDir, fmt.Sprintf("%s.json", cp.CheckpointID))
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return "", err
	}
	tmpFile := file + ".tmp"
	if err := os.WriteFile(tmpFile, append(data, '\n'), 0600); err != nil {
		return "", err
	}
	if err := os.Rename(tmpFile, file); err != nil {
		os.Remove(tmpFile)
		return "", err
	}
	return file, nil
}

// RecordLineage appends a lineage entry to persistent records.
func RecordLineage(rec LineageRecord) error {
	dataDir, err := config.DataDir()
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), "ai-control")
	}
	lineageDir := filepath.Join(dataDir, "lineage")
	_ = os.MkdirAll(lineageDir, 0700)

	file := filepath.Join(lineageDir, fmt.Sprintf("%s.json", rec.LineageID))
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := file + ".tmp"
	if err := os.WriteFile(tmpFile, append(data, '\n'), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, file); err != nil {
		os.Remove(tmpFile)
		return err
	}
	return nil
}
