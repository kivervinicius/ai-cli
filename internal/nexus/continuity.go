package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type continuityLaunch struct {
	Args              []string
	ProviderSessionID string
	Status            string
}

// continuityForNextGeneration decides whether a new runtime can truthfully
// resume the previous provider session. Cross-provider or unverifiable changes
// are NEW_SESSION; they are never labelled as resume/context recovery.
func continuityForNextGeneration(ctx context.Context, cfg AgentConfig, previous *store.RuntimeGeneration) (continuityLaunch, error) {
	if previous == nil || previous.RuntimeID == "" {
		return continuityLaunch{Status: store.ContinuityNewSession}, nil
	}
	if cfg.Provider != previous.Provider || cfg.Profile != previous.Profile || strings.EqualFold(cfg.ContinuityPolicy, "new_session") {
		return continuityLaunch{Status: store.ContinuityNewSession}, nil
	}
	if strings.TrimSpace(previous.ProviderSession) == "" {
		if strings.EqualFold(cfg.ContinuityPolicy, "native") {
			return continuityLaunch{}, fmt.Errorf("native continuity requested but provider session ID is unavailable")
		}
		return continuityLaunch{Status: store.ContinuityNewSession}, nil
	}
	d, err := driver.DefaultRegistry().Get(cfg.Provider)
	if err != nil {
		return continuityLaunch{}, err
	}
	profile := model.Profile{Name: cfg.Profile, Provider: cfg.Provider}
	can, reason := d.CanResume(ctx, profile, previous.ProviderSession)
	if !can {
		if strings.EqualFold(cfg.ContinuityPolicy, "native") {
			return continuityLaunch{}, fmt.Errorf("native continuity unavailable: %s", reason)
		}
		return continuityLaunch{Status: store.ContinuityNewSession}, nil
	}
	args, err := d.BuildResumeArgs(ctx, profile, previous.ProviderSession)
	if err != nil {
		if strings.EqualFold(cfg.ContinuityPolicy, "native") {
			return continuityLaunch{}, fmt.Errorf("build native resume: %w", err)
		}
		return continuityLaunch{Status: store.ContinuityNewSession}, nil
	}
	return continuityLaunch{
		Args:              args,
		ProviderSessionID: previous.ProviderSession,
		Status:            store.ContinuityNativeResumeUnverified,
	}, nil
}
