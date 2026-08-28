package driver

import (
	"context"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// ControlCapabilities defines granular runtime supervision capabilities supported by a provider driver.
type ControlCapabilities struct {
	Process          bool `json:"process"`
	Terminal         bool `json:"terminal"`
	Attach           bool `json:"attach"`
	StructuredEvents bool `json:"structured_events"`
	Sessions         bool `json:"sessions"`
	Resume           bool `json:"resume"`
	Fork             bool `json:"fork"`
	SubmitPrompt     bool `json:"submit_prompt"`
	CancelTurn       bool `json:"cancel_turn"`
	Approvals        bool `json:"approvals"`
	NativeUIAttach   bool `json:"native_ui_attach"`
	Headless         bool `json:"headless"`
	SlashControl     bool `json:"slash_control"`
}

// ControlDriver defines the interface for creating supervised runtime configurations for a provider.
type ControlDriver interface {
	ProviderID() string
	Detect(ctx context.Context) (model.DetectionResult, error)
	Capabilities(ctx context.Context, p model.Profile) ControlCapabilities
	BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (binary string, args []string, env []string, err error)
}
