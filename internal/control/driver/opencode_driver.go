package driver

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

type OpenCodeDriver struct{}

func NewOpenCodeDriver() *OpenCodeDriver { return &OpenCodeDriver{} }

func (d *OpenCodeDriver) ProviderID() string { return "opencode" }

func (d *OpenCodeDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := runtime.LookPath("opencode")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "opencode binary not found in PATH"}, nil
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}, nil
}

func (d *OpenCodeDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return ControlCapabilities{
		Process:          true,
		Terminal:         true,
		Attach:           true,
		StructuredEvents: true,
		Sessions:         true,
		Resume:           true,
		Fork:             false,
		SubmitPrompt:     true,
		CancelTurn:       true,
		Approvals:        false,
		NativeUIAttach:   false,
		Headless:         true,
		SlashControl:     true,
	}
}

func (d *OpenCodeDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("opencode")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("opencode", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":          home,
		"XDG_DATA_HOME": filepath.Join(home, ".local", "share"),
		"OPENCODE_HOME": home,
		"AI_PROFILE":    p.Name,
		"AI_PROVIDER":   "opencode",
	})

	return bin, extraArgs, env, nil
}
