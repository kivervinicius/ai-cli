//go:build windows

package main

import (
	"runtime"
)

func readOSMetrics(m *platformMetrics) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.RSSBytes = mem.Sys
}
