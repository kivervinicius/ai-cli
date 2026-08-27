package profile

import (
	"context"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
)

type AccountInfo = model.AccountInfo

// GetAccountInfo extracts authentic non-secret account identity, email, and subscription plan.
func GetAccountInfo(providerName, name string) model.AccountInfo {
	ctx := context.Background()
	p := model.Profile{Provider: providerName, Name: name}

	switch providerName {
	case "codex":
		return codex.New().InspectAuth(ctx, p)
	case "agy":
		return agy.New().InspectAuth(ctx, p)
	case "claude":
		return claude.New().InspectAuth(ctx, p)
	case "opencode":
		return opencode.New().InspectAuth(ctx, p)
	case "gemini":
		return gemini.New().InspectAuth(ctx, p)
	default:
		return model.AccountInfo{
			Status:        "Unknown provider",
			Health:        model.HealthUnknown,
			Authenticated: false,
		}
	}
}
