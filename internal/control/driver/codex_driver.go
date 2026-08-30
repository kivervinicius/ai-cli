package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/control/terminal"
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
	out, err := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	if err != nil || strings.Contains(out, "No such file or directory") {
		return model.DetectionResult{
			Installed:  false,
			BinaryPath: bin,
			Error:      strings.TrimSpace(out),
		}, nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    lines[0],
	}, nil
}

func (d *CodexDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
	det, _ := d.Detect(ctx)
	version := det.Version

	return EffectiveCapabilities{
		Process: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "os/exec",
			Tested:          true,
		},
		Terminal: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       terminal.BackendMechanism(),
			Tested:          true,
		},
		Attach: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "Nexus Control IPC Socket/Pipe",
			Tested:          true,
		},
		StructuredEvents: CapabilityEvidence{
			Status:          CapabilityUnsupported,
			ProviderVersion: version,
			Mechanism:       "codex app-server adapter",
			Reason:          "Structured app-server adapter disabled; running in truthful TERMINAL mode",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "~/.codex/sessions index",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "codex resume <session-id>",
			Reason:          "resume command supported by signature; not runtime-verified against a live codex session",
			Tested:          false,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Codex CLI does not support session branching/forking",
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Structured prompt submission requires app-server adapter; use terminal input",
			Tested: false,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C signal passthrough",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Structured approvals require app-server adapter; interact via terminal prompt",
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Native GUI attach not implemented for Codex",
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status: CapabilityPartial,
			Reason: "Supports -p non-interactive prompts in classic mode",
			Tested: true,
		},
		AutonomousCoding: CapabilityEvidence{Status: CapabilityUnsupported, Reason: "submit_prompt/headless autonomous contract is not verified", Tested: true},
		ReadOnlyReview:   CapabilityEvidence{Status: CapabilityUnsupported, Reason: "no verified read-only autonomous review mode", Tested: true},
		SlashControl: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Universal /ai slash command router",
			Tested:    true,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *CodexDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
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
		"HOME":             home,
		"CODEX_HOME":       home,
		"CODEX_CONFIG_DIR": home,
		"AI_PROFILE":       p.Name,
		"AI_PROVIDER":      "codex",
		"PATH":             runtime.EnhancedPATH(filepath.Dir(bin)),
	})

	return bin, extraArgs, env, nil
}

func (d *CodexDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *CodexDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume codex session without valid session ID")
	}
	return []string{"resume", providerSessionID}, nil
}

func (d *CodexDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return []string{"-m", "gpt-5.6-sol", kickoffPrompt}, nil
}

func (d *CodexDriver) BuildAutonomousArgs(ctx context.Context, p model.Profile, kickoffPrompt string, mode AutonomousMode, policy AutonomousPolicy) ([]string, error) {
	return nil, fmt.Errorf("codex autonomous execution is disabled until submit_prompt/headless support is verified")
}
