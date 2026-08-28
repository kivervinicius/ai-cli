package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"sync"
)

var (
	localMachineID string
	machineOnce    sync.Once
)

// LocalMachineID returns a deterministic unique identifier for the current machine.
func LocalMachineID() string {
	machineOnce.Do(func() {
		// 1. Try reading /etc/machine-id (Linux)
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				localMachineID = id
				return
			}
		}

		// 2. Try reading /var/lib/dbus/machine-id (Linux fallback)
		if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				localMachineID = id
				return
			}
		}

		// 3. Fallback to SHA256 of hostname
		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			hostname = "ai-control-node"
		}
		hash := sha256.Sum256([]byte(hostname))
		localMachineID = hex.EncodeToString(hash[:16])
	})
	return localMachineID
}
