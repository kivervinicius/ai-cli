//go:build !windows

package registry

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// IsProcessAlive checks whether a process with the given PID is currently running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, signal 0 tests process existence
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// getBootTime returns system boot time in seconds since epoch.
func getBootTime() int64 {
	data, err := os.ReadFile("/proc/stat")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "btime ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if bt, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						return bt
					}
				}
			}
		}
	}
	return 0
}

// IsProcessAliveWithGeneration checks whether a process is alive and its creation time
// roughly matches the recorded hostGeneration (to detect PID recycling).
func IsProcessAliveWithGeneration(pid int, hostGeneration int64) bool {
	if !IsProcessAlive(pid) {
		return false
	}

	if hostGeneration <= 0 {
		return true
	}

	// Read /proc/<pid>/stat field 22 (start time in clock ticks)
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err == nil {
		statStr := string(data)
		rParen := strings.LastIndex(statStr, ")")
		if rParen != -1 {
			parts := strings.Fields(statStr[rParen+1:])
			if len(parts) >= 20 { // field 2 is right after ')', so start time (22) is index 19
				if startTimeTicks, err := strconv.ParseInt(parts[19], 10, 64); err == nil {
					// Assume USER_HZ is 100 for simplicity (standard on Linux)
					bootTimeSec := getBootTime()
					if bootTimeSec > 0 {
						procStartSec := bootTimeSec + (startTimeTicks / 100)
						procStartNano := procStartSec * 1e9
						
						// If process was created more than 10 seconds after our recorded hostGeneration,
						// it is very likely a recycled PID.
						if procStartNano > hostGeneration+(10*1e9) {
							return false
						}
					}
				}
			}
		}
	}

	return true
}
