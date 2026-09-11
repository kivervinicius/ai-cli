package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// hasProviderLaunchFlag recognizes Nexus-owned launch flags before provider
// arguments are normalized. Provider CLIs must never receive these flags.
func hasProviderLaunchFlag(args []string, wanted string) bool {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), wanted) {
			return true
		}
	}
	return false
}

func removeProviderLaunchFlag(args []string, wanted string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), wanted) {
			continue
		}
		out = append(out, arg)
	}
	return out
}

// executeProviderSupervised launches the provider through the same persistent
// SessionHost used by Web/Desktop. This opt-in path gives terminal-only users
// slash control, detach/attach, handoff and generation tracking without
// requiring the Web or Desktop surface.
func executeProviderSupervised(provName, explicitProfile string, args []string) error {
	if strings.TrimSpace(provName) == "" {
		return fmt.Errorf("provider is required for supervised launch")
	}

	profiles, err := profile.List()
	if err != nil {
		return err
	}
	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	quotaEngine := quota.NewEngine(quota.DefaultTTL)
	for _, candidate := range profiles {
		if candidate.Provider != provName || candidate.Disabled {
			continue
		}
		candidates = append(candidates, candidate)
		account := profile.GetAccountInfo(provName, candidate.Name)
		snapshot := profile.GetUsageSnapshot(provName, candidate.Name)
		account.Usage = snapshot
		accounts[candidate.Name] = account
		if snapshot.Status != model.UsageUnknown && snapshot.Status != model.UsageError {
			if account.AccountScope.Verifiable() {
				_ = quotaEngine.SaveUsageForScope(account.AccountScope, snapshot)
			}
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no enabled profiles configured for provider %s", provName)
	}

	selected := explicitProfile
	if selected == "" {
		cfg, _ := config.LoadConfig()
		selector := scheduler.NewSelector(cfg, quotaEngine, cooldown.NewTracker())
		result, selectErr := selector.SelectBestProfile(context.Background(), provName, mustWorkingDirectory(), candidates, accounts, nil)
		if selectErr == nil && result != nil && result.SelectedProfile != nil {
			selected = result.SelectedProfile.Name
		}
	}
	if selected == "" {
		selected = candidates[0].Name
	}
	if !profileInCandidates(candidates, selected) {
		return fmt.Errorf("profile %q is not configured for provider %s", selected, provName)
	}

	workspace := mustWorkingDirectory()
	session, err := launcher.Default().Launch(context.Background(), launcher.LaunchOptions{
		ProviderID: provName,
		ProfileID:  selected,
		Workspace:  workspace,
		Args:       args,
		Standalone: false,
		Timeout:    15 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("supervised %s launch failed: %w", provName, err)
	}
	fmt.Fprintf(os.Stderr, "⚡ [nexus] supervised %s:%s runtime=%s\n", provName, selected, session.RuntimeID)
	return attachRuntime(session.RuntimeID)
}

func profileInCandidates(candidates []model.Profile, name string) bool {
	for _, candidate := range candidates {
		if candidate.Name == name {
			return true
		}
	}
	return false
}

func mustWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil || strings.TrimSpace(cwd) == "" {
		return "."
	}
	return cwd
}
