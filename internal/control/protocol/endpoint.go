package protocol

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

// EndpointPath returns the canonical local socket / pipe path for a runtime.
func EndpointPath(runtimeID string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\ai-control-%s`, runtimeID)
	}
	// Linux / macOS: use /tmp/ai-control-<uid>/<runtime-id>.sock or user runtime dir
	uid := os.Getuid()
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("ai-control-%d", uid))
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, fmt.Sprintf("%s.sock", runtimeID))
}

// Listen creates the local server listener for a runtime.
func Listen(runtimeID string) (net.Listener, error) {
	path := EndpointPath(runtimeID)
	if runtime.GOOS != "windows" {
		// Clean up existing socket file if dead
		_ = os.Remove(path)
		_ = os.MkdirAll(filepath.Dir(path), 0700)
		return net.Listen("unix", path)
	}
	// Windows fallback listener (TCP loopback or pipe wrapper)
	return net.Listen("tcp", "127.0.0.1:0")
}

// Dial connects to a running SessionHost endpoint.
func Dial(runtimeID string) (net.Conn, error) {
	path := EndpointPath(runtimeID)
	if runtime.GOOS != "windows" {
		return net.Dial("unix", path)
	}
	return net.Dial("tcp", path)
}
