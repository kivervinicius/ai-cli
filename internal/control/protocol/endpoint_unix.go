//go:build !windows

package protocol

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Listen creates the local server listener for a runtime.
func Listen(runtimeID string) (net.Listener, error) {
	path := EndpointPath(runtimeID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove dead/stale socket file if present
	_ = os.Remove(path)

	oldMask := syscall.Umask(0177)
	l, err := net.Listen("unix", path)
	syscall.Umask(oldMask)
	if err != nil {
		return nil, err
	}

	// Enforce user-only permissions (0600) on the unix socket file
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to chmod socket: %w", err)
	}

	return l, nil
}

// Dial connects to a running SessionHost endpoint.
func Dial(runtimeID string) (net.Conn, error) {
	path := EndpointPath(runtimeID)
	return net.DialTimeout("unix", path, 2*time.Second)
}

func isOwnedBy(st os.FileInfo, uid int) bool {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return sys.Uid == uint32(uid)
	}
	return false
}
