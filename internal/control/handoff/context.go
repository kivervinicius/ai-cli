package handoff

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/host"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// ContextEnvelope captures safe, transferable workspace state across different AI providers.
// Secrets, API keys, and private tokens are automatically redacted.
type ContextEnvelope struct {
	SchemaVersion   int       `json:"schema_version"`
	SourceProvider  string    `json:"source_provider"`
	SourceProfile   string    `json:"source_profile"`
	SourceSessionID string    `json:"source_session_id,omitempty"`
	Workspace       string    `json:"workspace"`
	Goal            string    `json:"goal,omitempty"`
	GitBranch       string    `json:"git_branch,omitempty"`
	GitStatus       string    `json:"git_status,omitempty"`
	ChangedFiles    []string  `json:"changed_files,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ExtractContextEnvelope inspects the current workspace and compiles a redacted context envelope.
func ExtractContextEnvelope(workspace, sourceProvider, sourceProfile, sourceSessionID, goal string) ContextEnvelope {
	env := ContextEnvelope{
		SchemaVersion:   1,
		SourceProvider:  sourceProvider,
		SourceProfile:   sourceProfile,
		SourceSessionID: sourceSessionID,
		Workspace:       workspace,
		Goal:            security.Redact(goal),
		CreatedAt:       time.Now(),
	}

	// 1. Get git branch
	if out, err := exec.Command("git", "-C", workspace, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		env.GitBranch = strings.TrimSpace(string(out))
	}

	// 2. Get git status short
	if out, err := exec.Command("git", "-C", workspace, "status", "--short").Output(); err == nil {
		rawStatus := string(out)
		env.GitStatus = security.Redact(rawStatus)

		for _, line := range strings.Split(rawStatus, "\n") {
			line = strings.TrimSpace(line)
			if len(line) > 3 {
				env.ChangedFiles = append(env.ChangedFiles, strings.TrimSpace(line[3:]))
			}
		}
	}

	return env
}

// FormatKickoffPrompt produces a clean, honest initial prompt for the target provider.
func (env ContextEnvelope) FormatKickoffPrompt() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== AI Control Context Handoff (from %s:%s) ===\n", strings.ToUpper(env.SourceProvider), env.SourceProfile))
	sb.WriteString(fmt.Sprintf("Workspace: %s\n", env.Workspace))
	if env.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("Active Git Branch: %s\n", env.GitBranch))
	}
	if len(env.ChangedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Modified Files (%d):\n", len(env.ChangedFiles)))
		for _, f := range env.ChangedFiles {
			sb.WriteString(fmt.Sprintf(" - %s\n", f))
		}
	}
	if env.Goal != "" {
		sb.WriteString(fmt.Sprintf("Current Goal / Task: %s\n", env.Goal))
	}
	sb.WriteString("==================================================\n")
	sb.WriteString("Please inspect the modified files and continue the ongoing task in this workspace.")
	return sb.String()
}

// PerformContextHandoff creates a new session on a DIFFERENT provider using the extracted context envelope.
func PerformContextHandoff(ctx context.Context, sourceRuntimeID, targetProvider, targetProfile string) (*registry.RuntimeSession, error) {
	reg := registry.DefaultRegistry()
	source, ok := reg.Get(sourceRuntimeID)
	if !ok {
		return nil, fmt.Errorf("source runtime %q not found", sourceRuntimeID)
	}

	// Resolve target profile if omitted
	if targetProfile == "" {
		profs, _ := profile.List()
		for _, p := range profs {
			if p.Provider == targetProvider {
				targetProfile = p.Name
				break
			}
		}
		if targetProfile == "" {
			targetProfile = "default"
		}
	}

	// 1. Extract context envelope
	env := ExtractContextEnvelope(source.Workspace, source.ProviderID, source.ProfileID, source.ProviderSessionID, "")
	kickoffPrompt := env.FormatKickoffPrompt()

	// 2. Stop source runtime
	client, err := protocol.NewClient(sourceRuntimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
	_ = reg.UpdateState(sourceRuntimeID, registry.StateHandoff)

	// 3. Get target driver
	d, err := driver.DefaultRegistry().Get(targetProvider)
	if err != nil {
		return nil, err
	}

	var extraArgs []string
	switch targetProvider {
	case "claude":
		extraArgs = []string{"-p", kickoffPrompt}
	case "codex":
		extraArgs = []string{"-m", "gpt-5", kickoffPrompt}
	default:
		extraArgs = []string{kickoffPrompt}
	}

	bin, args, envVars, err := d.BuildCommand(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, extraArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for target provider %s: %w", targetProvider, err)
	}

	newRuntimeID := fmt.Sprintf("%s-handoff-%d", targetProvider, len(reg.List())+1)
	newSession := registry.RuntimeSession{
		RuntimeID:       newRuntimeID,
		ProviderID:      targetProvider,
		ProfileID:       targetProfile,
		Workspace:       source.Workspace,
		State:           registry.StateStarting,
		ControlLevel:    registry.ControlLevelTerminal,
		ParentRuntimeID: source.RuntimeID,
		HandoffType:     "context",
	}

	sh, err := host.NewSessionHost(host.Config{
		Session: newSession,
		Binary:  bin,
		Args:    args,
		Env:     envVars,
		Cwd:     source.Workspace,
		UsePTY:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SessionHost for context handoff: %w", err)
	}

	if err := sh.Start(); err != nil {
		return nil, fmt.Errorf("failed to start context handoff runtime: %w", err)
	}

	_ = reg.Register(newSession)
	return &newSession, nil
}
