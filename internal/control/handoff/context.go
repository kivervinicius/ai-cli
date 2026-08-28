package handoff

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/host"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// FormatKickoffPrompt produces a clean, honest initial prompt for the target provider from a WorkCheckpoint.
func FormatKickoffPrompt(cp WorkCheckpoint) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== AI Control Context Handoff (from %s:%s) ===\n", strings.ToUpper(cp.SourceProvider), cp.SourceProfile))
	sb.WriteString(fmt.Sprintf("Workspace: %s\n", cp.Workspace))
	if cp.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("Active Git Branch: %s\n", cp.GitBranch))
	}
	if len(cp.ChangedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Modified Files (%d):\n", len(cp.ChangedFiles)))
		for _, f := range cp.ChangedFiles {
			sb.WriteString(fmt.Sprintf(" - %s\n", f))
		}
	}
	if cp.GitDiffStat != "" {
		sb.WriteString("Git Diff Summary:\n" + cp.GitDiffStat + "\n")
	}
	if cp.Goal != "" {
		sb.WriteString(fmt.Sprintf("Current Goal / Task: %s\n", cp.Goal))
	}
	sb.WriteString("==================================================\n")
	sb.WriteString("Please inspect the modified files and continue the ongoing task in this workspace.")
	return security.Redact(sb.String())
}

// PerformContextHandoff creates a new session on a DIFFERENT provider using a captured WorkCheckpoint.
func PerformContextHandoff(ctx context.Context, sourceRuntimeID, targetProvider, targetProfile string) (*registry.RuntimeSession, error) {
	reg := registry.DefaultRegistry()
	source, ok := reg.Get(sourceRuntimeID)
	if !ok {
		return nil, fmt.Errorf("source runtime %q not found", sourceRuntimeID)
	}

	// 1. Resolve Target Profile using Smart Account Selector if unspecified
	if targetProfile == "" {
		cfg, _ := config.LoadConfig()
		qEng := quota.NewEngine(5 * time.Minute)
		cdTracker := cooldown.NewTracker()
		sel := scheduler.NewSelector(cfg, qEng, cdTracker)

		allProfiles, _ := profile.List()
		var candidates []model.Profile
		accounts := make(map[string]model.AccountInfo)
		for _, p := range allProfiles {
			if p.Provider == targetProvider {
				candidates = append(candidates, p)
				accounts[p.Name] = profile.GetAccountInfo(targetProvider, p.Name)
			}
		}

		if len(candidates) > 0 {
			res, _ := sel.SelectBestProfile(ctx, targetProvider, source.Workspace, candidates, accounts, nil)
			if res.SelectedProfile != nil && res.SelectedProfile.Name != "" {
				targetProfile = res.SelectedProfile.Name
			} else {
				targetProfile = candidates[0].Name
			}
		} else {
			targetProfile = "default"
		}
	}

	// 2. Validate Target Authentication
	info := profile.GetAccountInfo(targetProvider, targetProfile)
	if !info.Authenticated {
		return nil, fmt.Errorf("target provider profile %s:%s is not authenticated", targetProvider, targetProfile)
	}

	// 3. Capture Safe Work Checkpoint
	cp := CaptureWorkCheckpoint(source.Workspace, source.RuntimeID, source.ProviderID, source.ProfileID, source.ProviderSessionID, "")
	if _, err := SaveCheckpoint(cp); err != nil {
		slog.Warn("Failed to save checkpoint during context handoff", "err", err)
	}

	kickoffPrompt := FormatKickoffPrompt(cp)

	// 4. Get Target Driver & Build Kickoff Command
	d, err := driver.DefaultRegistry().Get(targetProvider)
	if err != nil {
		return nil, fmt.Errorf("target driver error: %w", err)
	}

	extraArgs, err := d.BuildKickoffArgs(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, kickoffPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to construct kickoff arguments: %w", err)
	}

	bin, args, envVars, err := d.BuildCommand(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, extraArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for target provider %s: %w", targetProvider, err)
	}

	// 5. Launch Target Supervised Runtime FIRST
	newRuntimeID := fmt.Sprintf("%s-continue-%d", targetProvider, len(reg.List())+1)
	lineageID := fmt.Sprintf("lin-ctx-%s-%d", targetProvider, time.Now().UnixNano())

	newSession := registry.RuntimeSession{
		RuntimeID:         newRuntimeID,
		ProviderID:        targetProvider,
		ProfileID:         targetProfile,
		ProviderSessionID: "", // New session is a new thread, not an old session ID
		Workspace:         source.Workspace,
		State:             registry.StateStarting,
		ControlLevel:      registry.ControlLevelTerminal,
		ParentRuntimeID:   source.RuntimeID,
		HandoffType:       "context",
		LineageID:         lineageID,
		StartedAt:         time.Now(),
	}

	sh, err := host.NewSessionHost(host.Config{
		Session: newSession,
		Binary:  bin,
		Args:    args,
		Env:     envVars,
		Cwd:     source.Workspace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SessionHost for context handoff: %w", err)
	}

	if err := sh.Start(); err != nil {
		return nil, fmt.Errorf("failed to start context handoff runtime: %w", err)
	}

	// 6. Target started successfully. Now gracefullly stop source runtime.
	client, err := protocol.NewClient(sourceRuntimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
	_ = reg.UpdateState(sourceRuntimeID, registry.StateHandoff)

	_ = reg.Register(newSession)

	// 7. Record Lineage
	if err := RecordLineage(LineageRecord{
		LineageID:               lineageID,
		SourceRuntimeID:         source.RuntimeID,
		SourceProviderSessionID: source.ProviderSessionID,
		TargetRuntimeID:         newSession.RuntimeID,
		Type:                    "CONTEXT_HANDOFF",
		CreatedAt:               time.Now(),
		CheckpointID:            cp.CheckpointID,
	}); err != nil {
		slog.Warn("Failed to record lineage during context handoff", "err", err)
	}

	events.DefaultBus().Publish(events.NewEvent(
		newSession.RuntimeID,
		newSession.ProviderID,
		newSession.ProfileID,
		events.EventHandoffCompleted,
		fmt.Sprintf("Context handoff completed from %s to %s", source.RuntimeID, newSession.RuntimeID),
		map[string]any{"source_id": source.RuntimeID, "target_id": newSession.RuntimeID, "checkpoint_id": cp.CheckpointID},
	))

	return &newSession, nil
}
