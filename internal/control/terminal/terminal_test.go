//go:build !windows

package terminal

import (
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTerminalBackendExecution(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("echo", "hello world")

	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start terminal backend: %v", err)
	}
	defer backend.Close()

	if backend.PID() <= 0 {
		t.Errorf("expected positive PID, got %d", backend.PID())
	}

	buf := make([]byte, 1024)
	n, _ := backend.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "hello world") {
		// Try reading rest
		time.Sleep(50 * time.Millisecond)
		n2, _ := backend.Read(buf)
		output += string(buf[:n2])
	}

	if !strings.Contains(output, "hello") {
		t.Errorf("expected 'hello' in output, got %q", output)
	}

	_ = backend.Resize(30, 100)
}

func TestTerminalBackendStdin(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("cat")

	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start terminal backend: %v", err)
	}
	defer backend.Close()

	testInput := "streaming test line\n"
	_, err := backend.Write([]byte(testInput))
	if err != nil {
		t.Fatalf("failed to write to backend: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := backend.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("failed to read from backend: %v", err)
	}

	out := string(buf[:n])
	if !strings.Contains(out, "streaming test line") {
		t.Errorf("expected echo of input, got %q", out)
	}
}
