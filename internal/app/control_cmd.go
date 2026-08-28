package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/handoff"
	"github.com/kivervinicius/ai-cli/internal/control/host"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	controltui "github.com/kivervinicius/ai-cli/internal/control/tui"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

func controlCmd(args []string) error {
	if len(args) == 0 {
		target, err := controltui.RunControlTUI(context.Background())
		if err != nil {
			return err
		}
		if target != "" {
			return attachRuntime(target)
		}
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		controlHelp()
		return nil
	case "start":
		return controlStartCmd(args[1:])
	case "running", "list", "ls":
		return controlRunningCmd(args[1:])
	case "status":
		return controlStatusCmd(args[1:])
	case "attach":
		return controlAttachCmd(args[1:])
	case "stop":
		return controlStopCmd(args[1:])
	case "handoff":
		return controlHandoffCmd(args[1:])
	case "continue":
		return controlContinueCmd(args[1:])
	case "cleanup":
		return controlCleanupCmd(args[1:])
	case "doctor":
		return controlDoctorCmd(args[1:])
	default:
		return fmt.Errorf("unknown control command %q. Run 'ai control --help' for usage", args[0])
	}
}

func controlHelp() {
	fmt.Println(`AI Control Center — Supervised Agent Runtimes

USAGE:
  ai control [subcommand] [flags]
  ai ui

SUBCOMMANDS:
  (no args)                     Open interactive Bubble Tea Control Center
  start <provider> [--profile <name>] [args...]
                                Start a supervised runtime session
  running [--json]              List active managed runtimes
  status <runtime-id> [--json]  Display runtime details
  attach <runtime-id>           Connect terminal to running session
  stop <runtime-id>             Gracefully stop a runtime
  handoff <runtime-id> <provider:profile>
                                Same-provider account handoff
  continue <runtime-id> --with <provider[:profile]>
                                Cross-provider context handoff
  cleanup                       Clean up stale runtime records and dead sockets
  doctor [--json]               Audit control runtime environment and drivers

FLAGS:
  --json                        Output in machine-readable JSON format
  -h, --help                    Show this help message`)
}

func controlStartCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai control start <provider> [--profile <name>] [args...]")
	}

	providerID := args[0]
	var profileName string
	var extraArgs []string

	for i := 1; i < len(args); i++ {
		if (args[i] == "--profile" || args[i] == "-p") && i+1 < len(args) {
			profileName = args[i+1]
			i++
		} else {
			extraArgs = append(extraArgs, args[i])
		}
	}

	ctx := context.Background()

	// 1. Resolve Profile using Smart Account Selector if not explicitly given
	if profileName == "" {
		cwd, _ := os.Getwd()
		cfg, _ := config.LoadConfig()
		qEng := quota.NewEngine(5 * time.Minute)
		cdTracker := cooldown.NewTracker()
		sel := scheduler.NewSelector(cfg, qEng, cdTracker)

		allProfiles, _ := profile.List()
		var candidates []model.Profile
		accounts := make(map[string]model.AccountInfo)
		for _, p := range allProfiles {
			if p.Provider == providerID {
				candidates = append(candidates, p)
				accounts[p.Name] = profile.GetAccountInfo(providerID, p.Name)
			}
		}

		res, _ := sel.SelectBestProfile(ctx, providerID, cwd, candidates, accounts, nil)
		if res.SelectedProfile != nil && res.SelectedProfile.Name != "" {
			profileName = res.SelectedProfile.Name
		} else {
			def, _ := config.GetDefaultProfile(providerID)
			if def != "" {
				profileName = def
			} else {
				profileName = "default"
			}
		}
	}

	// 2. Resolve Control Driver
	d, err := driver.DefaultRegistry().Get(providerID)
	if err != nil {
		return err
	}

	p := model.Profile{Name: profileName, Provider: providerID}
	caps := d.Capabilities(ctx, p)

	bin, cmdArgs, env, err := d.BuildCommand(ctx, p, extraArgs)
	if err != nil {
		return fmt.Errorf("failed to build runtime command: %w", err)
	}

	cwd, _ := os.Getwd()
	reg := registry.DefaultRegistry()
	runtimeID := fmt.Sprintf("%s-%d", providerID, len(reg.List())+1)

	controlLevel := registry.ControlLevelTerminal
	if caps.StructuredEvents {
		controlLevel = registry.ControlLevelAPI
	}

	session := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   providerID,
		ProfileID:    profileName,
		Workspace:    cwd,
		State:        registry.StateStarting,
		ControlLevel: controlLevel,
		StartedAt:    time.Now(),
	}

	sh, err := host.NewSessionHost(host.Config{
		Session: session,
		Binary:  bin,
		Args:    cmdArgs,
		Env:     env,
		Cwd:     cwd,
		UsePTY:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to create SessionHost: %w", err)
	}

	if err := sh.Start(); err != nil {
		return fmt.Errorf("failed to start SessionHost: %w", err)
	}

	fmt.Printf("✓ Started supervised %s runtime (ID: %s, Profile: %s)\n", strings.ToUpper(providerID), runtimeID, profileName)
	fmt.Printf("  Connecting interactive terminal...\n\n")

	time.Sleep(100 * time.Millisecond)
	return attachRuntime(runtimeID)
}

func controlRunningCmd(args []string) error {
	jsonMode := hasFlag(args, "--json")
	reg := registry.DefaultRegistry()
	_, _ = reg.CleanupStale()
	active := reg.ListActive()

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(active)
	}

	if len(active) == 0 {
		fmt.Println("No active supervised runtimes. Start one with: ai control start <provider>")
		return nil
	}

	fmt.Printf("%-16s %-10s %-15s %-12s %-12s %s\n", "ID", "PROVIDER", "PROFILE", "STATE", "CONTROL", "WORKSPACE")
	fmt.Println(strings.Repeat("─", 80))
	for _, s := range active {
		fmt.Printf("%-16s %-10s %-15s %-12s %-12s %s\n",
			s.RuntimeID, strings.ToUpper(s.ProviderID), s.ProfileID, s.State, s.ControlLevel, s.Workspace,
		)
	}
	return nil
}

func controlStatusCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai control status <runtime-id> [--json]")
	}
	runtimeID := args[0]
	jsonMode := hasFlag(args, "--json")

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		// Fallback to registry
		reg := registry.DefaultRegistry()
		s, ok := reg.Get(runtimeID)
		if !ok {
			return fmt.Errorf("runtime %q not found or endpoint unreachable: %w", runtimeID, err)
		}
		if jsonMode {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(s)
		}
		fmt.Printf("Runtime ID:  %s\nProvider:    %s\nProfile:     %s\nState:       %s (Host unreachable)\n", s.RuntimeID, s.ProviderID, s.ProfileID, s.State)
		return nil
	}
	defer client.Close()

	st, err := client.Status()
	if err != nil {
		return err
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}

	fmt.Printf("Runtime ID:         %s\n", st.RuntimeID)
	fmt.Printf("Provider:           %s\n", strings.ToUpper(st.ProviderID))
	fmt.Printf("Profile:            %s\n", st.ProfileID)
	if st.ProviderSessionID != "" {
		fmt.Printf("Provider Session:   %s\n", st.ProviderSessionID)
	}
	fmt.Printf("PID:                %d\n", st.PID)
	fmt.Printf("State:              %s\n", st.State)
	fmt.Printf("Control Level:      %s\n", st.ControlLevel)
	fmt.Printf("Workspace:          %s\n", st.Workspace)
	fmt.Printf("Started At:         %s\n", st.StartedAt.Format(time.RFC3339))
	return nil
}

func controlAttachCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai control attach <runtime-id>")
	}
	return attachRuntime(args[0])
}

func attachRuntime(runtimeID string) error {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return fmt.Errorf("failed to attach to runtime %q: %w", runtimeID, err)
	}
	defer client.Close()

	// 1. Put user terminal into Raw Mode if stdin is a terminal
	isTerm := term.IsTerminal(int(os.Stdin.Fd()))
	if isTerm {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer func() {
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
			}()
		}

		// 2. Register window resize signal listener
		sigChan := make(chan os.Signal, 1)
		protocol.NotifyWinSizeChange(sigChan)
		defer signal.Stop(sigChan)

		go func() {
			for range sigChan {
				if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil && w > 0 && h > 0 {
					_ = client.Resize(h, w)
				}
			}
		}()

		// Send initial terminal window size
		if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil && w > 0 && h > 0 {
			_ = client.Resize(h, w)
		}
	}

	// 3. Send Attach command to switch connection to raw streaming
	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		return fmt.Errorf("attach failed: %w", err)
	}

	// Disable deadline after Attach RPC
	_ = client.ClearDeadline()

	// Print initial ring buffer history
	var history string
	if json.Unmarshal(resp.Data, &history) == nil && history != "" {
		os.Stdout.WriteString(history)
	}

	rawConn := client.RawConn()

	// 4. Stream stdout from runtime host to user terminal
	errChan := make(chan error, 2)
	go func() {
		// First flush any buffered bytes in client.reader!
		r := client.Reader()
		if r != nil && r.Buffered() > 0 {
			buf := make([]byte, r.Buffered())
			n, _ := r.Read(buf)
			if n > 0 {
				_, _ = os.Stdout.Write(buf[:n])
			}
		}
		_, copyErr := io.Copy(os.Stdout, rawConn)
		errChan <- copyErr
	}()

	// 5. Stream user stdin to runtime host
	go func() {
		_, copyErr := io.Copy(rawConn, os.Stdin)
		errChan <- copyErr
	}()

	<-errChan
	return nil
}

func controlStopCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ai control stop <runtime-id>")
	}
	runtimeID := args[0]
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return fmt.Errorf("failed to reach runtime %q: %w", runtimeID, err)
	}
	defer client.Close()

	if err := client.Stop(); err != nil {
		return err
	}
	fmt.Printf("✓ Sent stop signal to runtime %s\n", runtimeID)
	return nil
}

func controlHandoffCmd(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: ai control handoff <runtime-id> <provider:profile>")
	}
	runtimeID := args[0]
	target := args[1]

	newSess, err := handoff.PerformAccountHandoff(context.Background(), runtimeID, target)
	if err != nil {
		return fmt.Errorf("account handoff failed: %w", err)
	}

	fmt.Printf("✓ Account handoff completed: %s -> %s (New Runtime: %s)\n", runtimeID, target, newSess.RuntimeID)
	return attachRuntime(newSess.RuntimeID)
}

func controlContinueCmd(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: ai control continue <runtime-id> --with <provider[:profile]>")
	}
	runtimeID := args[0]
	var targetProvider, targetProfile string

	for i := 1; i < len(args); i++ {
		if (args[i] == "--with" || args[i] == "-w") && i+1 < len(args) {
			target := args[i+1]
			if idx := strings.Index(target, ":"); idx != -1 {
				targetProvider = target[:idx]
				targetProfile = target[idx+1:]
			} else {
				targetProvider = target
			}
			i++
		}
	}

	if targetProvider == "" {
		return errors.New("missing --with <provider[:profile]> argument")
	}

	newSess, err := handoff.PerformContextHandoff(context.Background(), runtimeID, targetProvider, targetProfile)
	if err != nil {
		return fmt.Errorf("context handoff failed: %w", err)
	}

	fmt.Printf("✓ Context handoff completed: %s -> %s:%s (New Runtime: %s)\n", runtimeID, targetProvider, targetProfile, newSess.RuntimeID)
	return attachRuntime(newSess.RuntimeID)
}

func controlCleanupCmd(args []string) error {
	cleaned, err := registry.DefaultRegistry().CleanupStale()
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	fmt.Printf("✓ Cleaned up %d stale runtime records.\n", cleaned)
	return nil
}

func controlDoctorCmd(args []string) error {
	jsonMode := hasFlag(args, "--json")
	ctx := context.Background()

	reg := registry.DefaultRegistry()
	staleCount, _ := reg.CleanupStale()

	drivers := driver.DefaultRegistry().List()
	type driverStatus struct {
		Provider   string `json:"provider"`
		Installed  bool   `json:"installed"`
		Version    string `json:"version,omitempty"`
		BinaryPath string `json:"binary_path,omitempty"`
		Error      string `json:"error,omitempty"`
	}

	var dStatuses []driverStatus
	for _, d := range drivers {
		det, _ := d.Detect(ctx)
		dStatuses = append(dStatuses, driverStatus{
			Provider:   d.ProviderID(),
			Installed:  det.Installed,
			Version:    det.Version,
			BinaryPath: det.BinaryPath,
			Error:      det.Error,
		})
	}

	report := map[string]any{
		"runtimes_total":  len(reg.List()),
		"runtimes_active": len(reg.ListActive()),
		"stale_cleaned":   staleCount,
		"drivers":         dStatuses,
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Println("=== AI Control Runtime Doctor ===")
	fmt.Printf("Active Runtimes:  %d\n", len(reg.ListActive()))
	fmt.Printf("Total Runtimes:   %d\n", len(reg.List()))
	fmt.Printf("Stale Cleaned:    %d\n\n", staleCount)

	fmt.Println("Provider Drivers Status:")
	for _, ds := range dStatuses {
		mark := "✓"
		if !ds.Installed {
			mark = "✗"
		}
		fmt.Printf(" %s %-10s (Installed: %t, Version: %s)\n", mark, strings.ToUpper(ds.Provider), ds.Installed, ds.Version)
	}

	return nil
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
