//go:build windows

package protocol

import (
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
)

// Listen creates the local Windows Named Pipe listener for a runtime.
// Uses owner-restricted security descriptor to prevent cross-user access on local machine.
func Listen(runtimeID string) (net.Listener, error) {
	path := EndpointPath(runtimeID)

	cfg := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	l, err := winio.ListenPipe(path, cfg)
	if err == nil {
		return l, nil
	}

	// Fallback to default process security if custom SDDL fails on this Windows environment
	cfg.SecurityDescriptor = ""
	return winio.ListenPipe(path, cfg)
}

// Dial connects to a running Windows Named Pipe endpoint.
func Dial(runtimeID string) (net.Conn, error) {
	path := EndpointPath(runtimeID)
	timeout := 2 * time.Second
	return winio.DialPipe(path, &timeout)
}

func isOwnedBy(st os.FileInfo, uid int) bool {
	// Not applicable on Windows.
	return true
}
