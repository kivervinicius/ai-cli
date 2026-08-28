package driver

import (
	"context"
	"os"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

type CodexDriver struct{}

func NewCodexDriver() *CodexDriver { return &CodexDriver{} }

func (d *CodexDriver) ProviderID() string { return "codex" }

func (d *CodexDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "codex binary not found in PATH"}, nil
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}, nil
}

func (d *CodexDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
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
		Approvals:        true,
		NativeUIAttach:   false,
		Headless:         true,
		SlashControl:     true,
	}
}

func (d *CodexDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("codex", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":        home,
		"CODEX_HOME":  home,
		"AI_PROFILE":  p.Name,
		"AI_PROVIDER": "codex",
	})

	return bin, extraArgs, env, nil
}
