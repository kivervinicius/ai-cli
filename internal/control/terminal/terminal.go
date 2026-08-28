package terminal

import (
	"io"
	"os/exec"
)

// Backend abstracts platform-specific terminal emulation (Unix PTY, Windows ConPTY/Pipes).
type Backend interface {
	io.Reader
	io.Writer
	io.Closer

	// Start launches the process wrapped in the platform terminal subsystem.
	Start(cmd *exec.Cmd, initialRows, initialCols int) error

	// Resize updates the terminal dimensions if supported by the backend.
	Resize(rows, cols int) error

	// PID returns the process ID of the supervised child process.
	PID() int

	// SupportsResize indicates whether dynamic window resizing is active.
	SupportsResize() bool

	// SupportsRawMode indicates whether VT/raw mode is supported.
	SupportsRawMode() bool
}

// NewBackend instantiates the optimal terminal backend for the current OS.
func NewBackend() Backend {
	return newPlatformBackend()
}
