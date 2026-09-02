package launcher

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/host"
	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/control/launchenv"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// LaunchOptions defines parameters for launching a supervised AI runtime.
type LaunchOptions struct {
	RuntimeID         string
	AgentID           string
	Title             string
	ProviderID        string
	ProfileID         string
	ProviderSessionID string
	Workspace         string
	Args              []string
	Standalone        bool // If true, runs host in-process instead of spawning detached daemon
	Timeout           time.Duration
	Model             string            `json:"model,omitempty"`
	Environment       map[string]string `json:"environment,omitempty"`
	Isolation         string            `json:"isolation,omitempty"`
	Options           map[string]any    `json:"options,omitempty"`
	PathPrepend       []string          `json:"path_prepend,omitempty"`
}

// Launcher unifies supervised SessionHost spawning and handshake verification across all commands.
type Launcher struct {
	reg     *registry.Registry
	drivers *driver.Registry
}

var defaultLauncher = NewLauncher()

func Default() *Launcher {
	return defaultLauncher
}

func NewLauncher() *Launcher {
	return &Launcher{
		reg:     registry.DefaultRegistry(),
		drivers: driver.DefaultRegistry(),
	}
}

// Launch allocates, starts, and verifies a supervised runtime.
func (l *Launcher) Launch(ctx context.Context, opts LaunchOptions) (*registry.RuntimeSession, error) {
	if opts.RuntimeID == "" {
		opts.RuntimeID = fmt.Sprintf("%s-%s", opts.ProviderID, ids.NewRuntimeID())
	}
	if opts.Workspace == "" {
		cwd, _ := os.Getwd()
		opts.Workspace = cwd
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	cleanupOrphanLaunchEnvelopes(10 * time.Minute)

	d, err := l.drivers.Get(opts.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("unknown driver for provider %q: %w", opts.ProviderID, err)
	}

	prof := model.Profile{
		Provider: opts.ProviderID,
		Name:     opts.ProfileID,
	}

	configuredArgs, err := driver.ApplyLaunchConfiguration(opts.ProviderID, opts.Model, opts.Options, opts.Args)
	if err != nil {
		return nil, fmt.Errorf("invalid launch configuration for %s:%s: %w", opts.ProviderID, opts.ProfileID, err)
	}
	bin, extraArgs, env, err := d.BuildCommand(ctx, prof, configuredArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for %s:%s: %w", opts.ProviderID, opts.ProfileID, err)
	}
	env = launchenv.Merge(env, opts.Environment, opts.PathPrepend)

	title := opts.Title
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("%s (%s)", strings.ToUpper(opts.ProviderID), opts.ProfileID)
	}

	// Detached hosts consume arguments through a one-time private envelope;
	// standalone hosts can keep them in memory without registry persistence.
	registryArgs := append([]string(nil), extraArgs...)
	if !opts.Standalone {
		registryArgs = nil
	}
	sess := registry.RuntimeSession{
		RuntimeID:         opts.RuntimeID,
		AgentID:           opts.AgentID,
		Title:             title,
		ProviderID:        opts.ProviderID,
		ProfileID:         opts.ProfileID,
		ProviderSessionID: opts.ProviderSessionID,
		Model:             opts.Model,
		Workspace:         opts.Workspace,
		Binary:            bin,
		Args:              registryArgs,
		Env:               env,
		State:             registry.StateStarting,
		ControlLevel:      d.EffectiveCaps(ctx, prof).ControlLevel,
		ControlEndpoint:   protocol.EndpointPath(opts.RuntimeID),
		StartedAt:         time.Now(),
		MachineID:         registry.LocalMachineID(),
		Location:          "local",
		Transport:         "ipc",
	}

	if err := l.reg.Register(sess); err != nil {
		return nil, fmt.Errorf("failed to register runtime session: %w", err)
	}

	if opts.Standalone {
		sh, err := host.NewSessionHost(host.Config{
			Session:     sess,
			Binary:      bin,
			Args:        extraArgs,
			Env:         env,
			Cwd:         opts.Workspace,
			InitialRows: 24,
			InitialCols: 80,
		})
		if err != nil {
			_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
			return nil, fmt.Errorf("failed to initialize SessionHost: %w", err)
		}
		if err := sh.Start(); err != nil {
			_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
			return nil, fmt.Errorf("failed to start SessionHost: %w", err)
		}
	} else {
		if err := createLaunchEnvelope(opts.RuntimeID, extraArgs); err != nil {
			_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
			return nil, fmt.Errorf("failed to create private launch envelope: %w", err)
		}
		selfExe, err := os.Executable()
		if err != nil {
			selfExe = "ai"
		}
		proc, err := SpawnDetachedHost(selfExe, opts.RuntimeID)
		if err != nil {
			removeLaunchEnvelope(opts.RuntimeID)
			_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
			return nil, fmt.Errorf("failed to spawn detached SessionHost daemon: %w", err)
		}
		sess.HostPID = proc.Pid
		_ = l.reg.Register(sess)
	}

	// Handshake: wait for IPC endpoint to be active and responsive
	if err := protocol.WaitForEndpoint(ctx, opts.RuntimeID, opts.Timeout); err != nil {
		_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
		return nil, fmt.Errorf("runtime handshake timed out for %s: %w", opts.RuntimeID, err)
	}

	// Verify handshake with status probe
	client, err := protocol.NewClient(opts.RuntimeID)
	if err != nil {
		_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
		return nil, fmt.Errorf("failed to connect to initialized runtime %s: %w", opts.RuntimeID, err)
	}
	defer client.Close()

	status, err := client.Status()
	if err != nil {
		_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
		return nil, fmt.Errorf("status handshake probe failed for runtime %s: %w", opts.RuntimeID, err)
	}

	sess.PID = status.PID
	sess.State = registry.StateRunning
	_ = l.reg.Register(sess)

	return &sess, nil
}
