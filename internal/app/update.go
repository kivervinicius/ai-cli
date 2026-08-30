package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
	"github.com/kivervinicius/ai-cli/internal/nexus"
)

const updateCommandTimeout = 30 * time.Second

func runUpdateCommand(binary string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateCommandTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, binary, args...).Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("timed out after %s", updateCommandTimeout)
		}
		return err
	}
	return nil
}

// UpdateResult summarizes the outcome of updating Nexus and Maestro.
type UpdateResult struct {
	NexusUpdated   bool   `json:"nexus_updated"`
	NexusVersion   string `json:"nexus_version"`
	MaestroUpdated bool   `json:"maestro_updated"`
	MaestroVersion string `json:"maestro_version"`
	Error          string `json:"error,omitempty"`
}

func updateCmd(args []string) error {
	result := PerformSystemUpdate()

	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Println("=== IAPro Nexus & Orquestrador Maestro Updater ===")
	if result.Error != "" {
		fmt.Printf("\n⚠️  Update completed with warning/error: %s\n", result.Error)
	} else {
		fmt.Println("\n✓ System is fully up to date!")
	}
	fmt.Printf("• Nexus: %s\n", result.NexusVersion)
	fmt.Printf("• Maestro: %s\n", result.MaestroVersion)

	return nil
}

// PerformSystemUpdate performs the update sequence for both Maestro and Nexus.
func PerformSystemUpdate() UpdateResult {
	res := UpdateResult{
		NexusVersion: buildinfo.Version,
	}

	// 1. Update Orquestrador Maestro
	if npmPath, err := exec.LookPath("npm"); err == nil {
		if err := runUpdateCommand(npmPath, "install", "-g", "@iapro/orquestrador-maestro-cli@latest"); err != nil {
			res.Error = fmt.Sprintf("failed to update @iapro/orquestrador-maestro-cli: %v", err)
		} else {
			res.MaestroUpdated = true
		}
	} else {
		res.Error = "npm was not found; Maestro was not updated"
	}

	// npm is the package authority for the Maestro CLI. Do not invoke its
	// interactive/update subcommand from a browser request: it may spawn an
	// installer process that outlives the HTTP request.
	if maestroBin, err := exec.LookPath("orquestrador-maestro"); err == nil {
		// Get new version
		cmdVer := exec.Command(maestroBin, "version")
		if out, err := cmdVer.Output(); err == nil {
			res.MaestroVersion = strings.TrimSpace(string(out))
		}
	} else {
		res.MaestroVersion = "not installed"
	}

	// 2. Ensure symlinks in ~/.local/bin
	targetDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	_ = os.MkdirAll(targetDir, 0755)

	if maestroBin, err := exec.LookPath("orquestrador-maestro"); err == nil {
		_ = os.Remove(filepath.Join(targetDir, "maestro"))
		_ = os.Remove(filepath.Join(targetDir, "orquestrador"))
		_ = os.Symlink(maestroBin, filepath.Join(targetDir, "maestro"))
		_ = os.Symlink(maestroBin, filepath.Join(targetDir, "orquestrador"))
	}

	// 3. Nexus binary updates need the release workflow. Do not report a
	// successful update when this command has not rebuilt and replaced it.
	mClient := nexus.NewMaestroClient()
	status := mClient.Status()
	if status.Capabilities != nil {
		res.MaestroVersion = status.Capabilities.Version
	}
	res.NexusUpdated = false
	const nexusNotice = "Nexus binary update was not performed; use the release workflow to build and install a new binary"
	if res.Error == "" {
		res.Error = nexusNotice
	} else {
		res.Error += "; " + nexusNotice
	}
	return res
}
