package main

import (
	"runtime"
)

type platformMetrics struct {
	Goroutines int     `json:"goroutines"`
	RSSBytes   uint64  `json:"rss_bytes"`
	Sockets    int     `json:"sockets"`
	CPUSeconds float64 `json:"cpu_seconds"`
}

func readMetrics() platformMetrics {
	metrics := platformMetrics{
		Goroutines: runtime.NumGoroutine(),
	}
	readOSMetrics(&metrics)
	return metrics
}
