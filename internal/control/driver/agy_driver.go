package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
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

func (d *AGYDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
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
			Reason:          "AGY CLI operates as standalone binary without remote daemon protocol",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "~/.gemini/antigravity-cli session transcript files",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "agy --conversation=<session-id>",
			Tested:          true,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "agy -p / --prompt",
			Tested:          true,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C signal passthrough",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "AGY tool approvals handled interactively in terminal",
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "agy -p (print mode)",
			Tested:          true,
		},
		SlashControl: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Universal /ai slash command router",
			Tested:    true,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *AGYDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *AGYDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	bin, err := runtime.LookPath("agy")
	if err != nil {
		return "", nil, nil, err
	}

	home, err := bootstrapProfile("agy", p)
	if err != nil {
		return "", nil, nil, err
	}
	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":                             home,
		"AI_PROFILE":                       p.Name,
		"AI_PROVIDER":                      "agy",
		"PYTHON_KEYRING_BACKEND":           "keyring.backends.null.Keyring",
		"AI_HOST_DBUS_SESSION_BUS_ADDRESS": os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
		"PATH":                             runtime.EnhancedPATH(filepath.Dir(bin)),
	}, "DBUS_SESSION_BUS_ADDRESS", "GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID")

	wrappedBin, wrappedArgs := runtime.WrapWithIsolatedSecretService(bin, extraArgs)
	return wrappedBin, wrappedArgs, env, nil
}

func (d *AGYDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *AGYDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume agy session without valid session ID")
	}
	return []string{"--conversation=" + providerSessionID}, nil
}

func (d *AGYDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return []string{"-p", kickoffPrompt}, nil
}
