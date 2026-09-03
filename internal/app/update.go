package app

import (
	"encoding/json"
	"fmt"

	"github.com/kivervinicius/ai-cli/internal/nexus"
)

// UpdateResult summarizes the outcome of updating Nexus and Maestro.
type UpdateResult = nexus.UpdateResult

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

// PerformSystemUpdate updates the Maestro library. The Nexus binary is not replaced.
func PerformSystemUpdate() UpdateResult {
	return nexus.PerformSystemUpdate()
}
