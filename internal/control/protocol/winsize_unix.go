//go:build !windows

package protocol

import (
	"os"
	"os/signal"
	"syscall"
)

// NotifyWinSizeChange registers a channel for window resize signals.
func NotifyWinSizeChange(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
