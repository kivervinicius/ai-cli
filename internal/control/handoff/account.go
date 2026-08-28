package handoff

import (
	"context"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/host"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// PerformAccountHandoff transitions active work to another profile of the SAME provider,
// preserving the underlying provider session ID (e.g. Codex thread / AGY conversation ID).
func PerformAccountHandoff(ctx context.Context, sourceRuntimeID, targetProfileName string) (*registry.RuntimeSession, error) {
	reg := registry.DefaultRegistry()
	source, ok := reg.Get(sourceRuntimeID)
	if !ok {
		return nil, fmt.Errorf("source runtime %q not found", sourceRuntimeID)
	}

	// Parse "provider:profile" or simple "profile"
	targetProfile := targetProfileName
	if idx := strings.Index(targetProfileName, ":"); idx != -1 {
		targetProfile = targetProfileName[idx+1:]
	}

	if targetProfile == source.ProfileID {
		return nil, fmt.Errorf("target profile is identical to current active profile %q", source.ProfileID)
	}

	// 1. Verify target profile exists and is authenticated
	info := profile.GetAccountInfo(source.ProviderID, targetProfile)
	if !info.Authenticated {
		return nil, fmt.Errorf("target profile %s:%s is not authenticated", source.ProviderID, targetProfile)
	}

	// 2. Verify driver support for resume
	d, err := driver.DefaultRegistry().Get(source.ProviderID)
	if err != nil {
		return nil, err
	}
	caps := d.Capabilities(ctx, model.Profile{Name: targetProfile, Provider: source.ProviderID})
	if !caps.Resume {
		return nil, fmt.Errorf("provider %s does not support cross-account session resumption", source.ProviderID)
	}

	// 3. Gracefully stop source runtime
	client, err := protocol.NewClient(sourceRuntimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
	_ = reg.UpdateState(sourceRuntimeID, registry.StateHandoff)

	// 4. Build resume arguments for target profile
	var resumeArgs []string
	if source.ProviderSessionID != "" {
		switch source.ProviderID {
		case "codex":
			resumeArgs = []string{"resume", source.ProviderSessionID}
		case "agy":
			resumeArgs = []string{"--conversation=" + source.ProviderSessionID}
		case "claude":
			resumeArgs = []string{"--resume", source.ProviderSessionID}
		case "opencode":
			resumeArgs = []string{"session", source.ProviderSessionID}
		}
	}

	bin, args, env, err := d.BuildCommand(ctx, model.Profile{Name: targetProfile, Provider: source.ProviderID}, resumeArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for target profile: %w", err)
	}

	// 5. Spawn new runtime session
	newRuntimeID := fmt.Sprintf("%s-handoff-%d", source.ProviderID, len(reg.List())+1)
	newSession := registry.RuntimeSession{
		RuntimeID:         newRuntimeID,
		ProviderID:        source.ProviderID,
		ProfileID:         targetProfile,
		ProviderSessionID: source.ProviderSessionID,
		Workspace:         source.Workspace,
		State:             registry.StateStarting,
		ControlLevel:      source.ControlLevel,
		ParentRuntimeID:   source.RuntimeID,
		HandoffType:       "account",
	}

	sh, err := host.NewSessionHost(host.Config{
		Session: newSession,
		Binary:  bin,
		Args:    args,
		Env:     env,
		Cwd:     source.Workspace,
		UsePTY:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SessionHost for handoff: %w", err)
	}

	if err := sh.Start(); err != nil {
		return nil, fmt.Errorf("failed to start handoff runtime: %w", err)
	}

	_ = reg.Register(newSession)
	return &newSession, nil
}
