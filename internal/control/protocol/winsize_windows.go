//go:build windows

package protocol

import "os"

// NotifyWinSizeChange is a no-op on Windows.
func NotifyWinSizeChange(ch chan os.Signal) {}
