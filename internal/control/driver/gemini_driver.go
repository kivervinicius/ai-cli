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

type GeminiDriver struct{}

func NewGeminiDriver() *GeminiDriver { return &GeminiDriver{} }

func (d *GeminiDriver) ProviderID() string { return "gemini" }

func (d *GeminiDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := runtime.LookPath("gemini")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "gemini binary not found in PATH"}, nil
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}, nil
}

func (d *GeminiDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return ControlCapabilities{
		Process:          true,
		Terminal:         true,
		Attach:           true,
		StructuredEvents: false,
		Sessions:         true,
		Resume:           false,
		Fork:             false,
		SubmitPrompt:     false,
		CancelTurn:       false,
		Approvals:        false,
		NativeUIAttach:   false,
		Headless:         false,
		SlashControl:     true,
	}
}

func (d *GeminiDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("gemini")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("gemini", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":             home,
		"GEMINI_CLI_HOME":  home,
		"AI_PROFILE":       p.Name,
		"AI_PROVIDER":      "gemini",
	})

	return bin, extraArgs, env, nil
}
