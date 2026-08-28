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

type AGYDriver struct{}

func NewAGYDriver() *AGYDriver { return &AGYDriver{} }

func (d *AGYDriver) ProviderID() string { return "agy" }

func (d *AGYDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := runtime.LookPath("agy")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "agy binary not found in PATH"}, nil
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}, nil
}

func (d *AGYDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return ControlCapabilities{
		Process:          true,
		Terminal:         true,
		Attach:           true,
		StructuredEvents: false,
		Sessions:         true,
		Resume:           true,
		Fork:             false,
		SubmitPrompt:     false,
		CancelTurn:       false,
		Approvals:        false,
		NativeUIAttach:   false,
		Headless:         false,
		SlashControl:     true,
	}
}

func (d *AGYDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("agy")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("agy", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":        home,
		"AI_PROFILE":  p.Name,
		"AI_PROVIDER": "agy",
	})

	return bin, extraArgs, env, nil
}
