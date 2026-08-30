package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

type ClaudeDriver struct{}

func NewClaudeDriver() *ClaudeDriver { return &ClaudeDriver{} }

func (d *ClaudeDriver) ProviderID() string { return "claude" }

func (d *ClaudeDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := runtime.LookPath("claude")
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "claude binary not found in PATH"}, nil
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

func (d *ClaudeDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
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
			Mechanism:       "PTY / ConPTY TerminalBackend",
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
			Reason:          "Claude Code runs interactively inside terminal without remote structured daemon",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "~/.claude session store",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "claude --resume <session-id>",
			Tested:          true,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "claude -p / print mode",
			Tested:          true,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C signal passthrough",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Claude handles approvals in interactive TUI prompt",
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "claude -p (print mode)",
			Tested:          true,
		},
		AutonomousCoding: CapabilityEvidence{Status: CapabilitySupported, ProviderVersion: version, Mechanism: "claude -p with explicit permission policy", Tested: true},
		ReadOnlyReview:   CapabilityEvidence{Status: CapabilitySupported, ProviderVersion: version, Mechanism: "claude --permission-mode plan", Tested: true},
		SlashControl: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Universal /ai slash command router",
			Tested:    true,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *ClaudeDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *ClaudeDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("claude")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("claude", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":              home,
		"CLAUDE_CONFIG_DIR": filepath.Join(home, ".claude"),
		"AI_PROFILE":        p.Name,
		"AI_PROVIDER":       "claude",
		"PATH":              runtime.EnhancedPATH(filepath.Dir(bin)),
	})

	return bin, extraArgs, env, nil
}

func (d *ClaudeDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *ClaudeDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume claude session without valid session ID")
	}
	return []string{"--resume", providerSessionID}, nil
}

func (d *ClaudeDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return []string{"-p", kickoffPrompt}, nil
}

func (d *ClaudeDriver) BuildAutonomousArgs(ctx context.Context, p model.Profile, kickoffPrompt string, mode AutonomousMode, policy AutonomousPolicy) ([]string, error) {
	args, err := d.BuildKickoffArgs(ctx, p, kickoffPrompt)
	if err != nil {
		return nil, err
	}
	switch mode {
	case AutonomousReview:
		return append(args, "--permission-mode", "plan", "--output-format", "text"), nil
	case AutonomousCoding:
		if !policy.AllowToolAutoApproval {
			return nil, fmt.Errorf("claude autonomous coding requires explicit tool auto approval")
		}
		return append(args, "--dangerously-skip-permissions", "--output-format", "text"), nil
	default:
		return nil, fmt.Errorf("unsupported autonomous mode %q", mode)
	}
}
