package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
	"github.com/kivervinicius/ai-cli/internal/nexus"
)

// UpdateResult summarizes the outcome of updating Nexus and Maestro.
type UpdateResult struct {
	NexusUpdated   bool   `json:"nexus_updated"`
	NexusVersion   string `json:"nexus_version"`
	MaestroUpdated bool   `json:"maestro_updated"`
	MaestroVersion string `json:"maestro_version"`
	Error          string `json:"error,omitempty"`
}

func updateCmd(args []string) error {
	fmt.Println("=== IAPro Nexus & Orquestrador Maestro Updater ===")
	fmt.Println("Checking and applying updates...")

	result := PerformSystemUpdate()

	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return nil
	}

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
	fmt.Println("\n[1/2] Updating Orquestrador Maestro...")
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "install", "-g", "@iapro/orquestrador-maestro-cli@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("  ⚠️ Failed to update @iapro/orquestrador-maestro-cli via npm: %v\n", err)
		} else {
			res.MaestroUpdated = true
			fmt.Println("  ✓ Updated @iapro/orquestrador-maestro-cli to latest release.")
		}
	}

	// Run maestro update to sync skills
	if maestroBin, err := exec.LookPath("orquestrador-maestro"); err == nil {
		cmdSync := exec.Command(maestroBin, "update", "--non-interactive")
		_ = cmdSync.Run()

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
	fmt.Println("\n[2/2] Checking IAPro Nexus binary update status...")
	mClient := nexus.NewMaestroClient()
	status := mClient.Status()
	if status.Capabilities != nil {
		res.MaestroVersion = status.Capabilities.Version
	}
	res.NexusUpdated = false
	res.Error = "Nexus binary update was not performed; use the release workflow to build and install a new binary"
	return res
}
