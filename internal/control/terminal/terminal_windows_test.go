//go:build windows

package terminal

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestConPTYBackendEcho verifies the real ConPTY backend end-to-end:
// spawns a process attached to a pseudo console and reads its output.
func TestConPTYBackendEcho(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("cmd.exe", "/C", "echo hello-conpty")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start ConPTY backend: %v", err)
	}
	defer backend.Close()

	if !backend.SupportsResize() {
		t.Errorf("ConPTY backend must support resize, got false")
	}
	if !backend.SupportsRawMode() {
		t.Errorf("ConPTY backend must support raw mode, got false")
	}
	if got := backend.Mechanism(); !strings.Contains(got, "ConPTY") {
		t.Errorf("expected ConPTY mechanism, got %q", got)
	}

	buf := make([]byte, 4096)
	deadline := time.Now().Add(10 * time.Second)
	var out strings.Builder
	for time.Now().Before(deadline) {
		n, err := backend.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if strings.Contains(out.String(), "hello-conpty") {
			break
		}
		if err == io.EOF {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(out.String(), "hello-conpty") {
		t.Fatalf("expected ConPTY output to contain hello-conpty, got %q", out.String())
	}

	// Resize round-trip must not error.
	if err := backend.Resize(30, 100); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	if err := backend.Wait(); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

// TestConPTYBackendInteractive verifies input is delivered to the child.
func TestConPTYBackendInteractive(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("cmd.exe", "/C", "set /p X=prompt: && echo GOT-%X%")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start ConPTY backend: %v", err)
	}
	defer backend.Close()

	// Wait for the prompt then answer.
	deadline := time.Now().Add(10 * time.Second)
	buf := make([]byte, 4096)
	var out strings.Builder
	answered := false
	for time.Now().Before(deadline) {
		n, err := backend.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if !answered && strings.Contains(out.String(), "prompt:") {
			if _, err := backend.Write([]byte("ABC\n")); err != nil {
				t.Fatalf("failed to write to ConPTY input: %v", err)
			}
			answered = true
		}
		if strings.Contains(out.String(), "GOT-ABC") {
			break
		}
		if err == io.EOF {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(out.String(), "GOT-ABC") {
		t.Fatalf("expected interactive echo GOT-ABC, got %q", out.String())
	}

	// Interrupt signal must not panic.
	_ = backend.Signal(os.Interrupt)
	_ = backend.Kill()
}
