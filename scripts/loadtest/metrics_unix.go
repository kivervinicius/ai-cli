//go:build unix || linux || darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func readOSMetrics(m *platformMetrics) {
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				var kilobytes uint64
				_, _ = fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "VmRSS:")), "%d kB", &kilobytes)
				m.RSSBytes = kilobytes * 1024
			}
		}
	}
	if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
		for _, entry := range entries {
			target, targetErr := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if targetErr == nil && strings.HasPrefix(target, "socket:") {
				m.Sockets++
			}
		}
	}
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err == nil {
		m.CPUSeconds = float64(usage.Utime.Sec) + float64(usage.Utime.Usec)/1e6 + float64(usage.Stime.Sec) + float64(usage.Stime.Usec)/1e6
	}
}
