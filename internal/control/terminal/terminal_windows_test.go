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

type conPTYRead struct {
	data []byte
	err  error
}

func readConPTYUntil(t *testing.T, backend Backend, match string, respond func()) string {
	t.Helper()
	reads := make(chan conPTYRead, 8)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := backend.Read(buf)
			chunk := append([]byte(nil), buf[:n]...)
			reads <- conPTYRead{data: chunk, err: err}
			if err != nil {
				return
			}
		}
	}()

	var out strings.Builder
	answered := false
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case result := <-reads:
			out.Write(result.data)
			if !answered && respond != nil && strings.Contains(out.String(), "prompt:") {
				respond()
				answered = true
			}
			if strings.Contains(out.String(), match) || result.err == io.EOF {
				return out.String()
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for %q; output=%q", match, out.String())
		}
	}
}

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

	out := readConPTYUntil(t, backend, "hello-conpty", nil)
	if !strings.Contains(out, "hello-conpty") {
		t.Fatalf("expected ConPTY output to contain hello-conpty, got %q", out)
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

	// Read asynchronously so a blocked pipe read cannot defeat the deadline.
	out := readConPTYUntil(t, backend, "GOT-ABC", func() {
		if _, err := backend.Write([]byte("ABC\n")); err != nil {
			t.Fatalf("failed to write to ConPTY input: %v", err)
		}
	})

	if !strings.Contains(out, "GOT-ABC") {
		t.Fatalf("expected interactive echo GOT-ABC, got %q", out)
	}

	// Interrupt signal must not panic.
	_ = backend.Signal(os.Interrupt)
	_ = backend.Kill()
}
