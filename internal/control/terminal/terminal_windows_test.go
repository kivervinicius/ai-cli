//go:build windows

package terminal

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCreateProcessSpecWrapsWindowsShellShim(t *testing.T) {
	cmd := exec.Command(`C:\Users\fbolz\AppData\Roaming\npm\codex.cmd`, "--help")
	app, line := createProcessSpec(cmd)

	if app != os.Getenv("ComSpec") {
		t.Fatalf("expected ComSpec application, got %q", app)
	}
	if !strings.Contains(line, " /d /s /c ") {
		t.Fatalf("expected cmd.exe shell flags, got %q", line)
	}
	if !strings.Contains(line, `codex.cmd`) || !strings.Contains(line, `--help`) {
		t.Fatalf("expected shim and arguments in command line, got %q", line)
	}
}

func TestUTF16EnvBlockSeparatesEntriesAndTerminatesOnce(t *testing.T) {
	block := utf16EnvBlock([]string{"FIRST=one", "SECOND=two"})
	first := syscall.StringToUTF16("FIRST=one")
	second := syscall.StringToUTF16("SECOND=two")
	if len(block) != len(first)+len(second)+1 {
		t.Fatalf("expected one terminator per entry plus final block terminator, got %d UTF-16 units", len(block))
	}
	for i, value := range first {
		if block[i] != value {
			t.Fatalf("first entry mismatch at %d: got %d, want %d", i, block[i], value)
		}
	}
	for i, value := range second {
		if block[len(first)+i] != value {
			t.Fatalf("second entry mismatch at %d: got %d, want %d", i, block[len(first)+i], value)
		}
	}
	if block[len(block)-1] != 0 {
		t.Fatal("environment block must end with an additional NUL")
	}
}

// TestWindowsBackendEcho verifies the Windows backend end-to-end.
func TestWindowsBackendEcho(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("cmd.exe")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start ConPTY backend: %v", err)
	}
	defer backend.Close()
	if _, err := backend.Write([]byte("echo hello-conpty\r\nexit\r\n")); err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	if strings.HasPrefix(backend.Mechanism(), "ConPTY") {
		if !backend.SupportsResize() || !backend.SupportsRawMode() {
			t.Fatalf("ConPTY backend must support resize and raw mode")
		}
	} else if backend.SupportsResize() || backend.SupportsRawMode() {
		t.Fatalf("standard-pipe backend must report unsupported resize/raw mode")
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
		t.Fatalf("expected Windows terminal output to contain hello-conpty, got %q", out.String())
	}

	// Resize round-trip must not error.
	if err := backend.Resize(30, 100); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	if err := backend.Wait(); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

// TestWindowsBackendInteractive verifies input is delivered to the child.
func TestWindowsBackendInteractive(t *testing.T) {
	backend := NewBackend()
	cmd := exec.Command("cmd.exe", "/C", "set /p X=prompt: && echo GOT-%X%")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start ConPTY backend: %v", err)
	}
	defer backend.Close()
	if !strings.HasPrefix(backend.Mechanism(), "ConPTY") {
		t.Skip("interactive line discipline requires ConPTY; standard pipes are validated by the echo test")
	}
	if _, err := backend.Write([]byte("ABC\r\n")); err != nil {
		t.Fatalf("failed to write interactive input: %v", err)
	}

	// Read the prompt and response without relying on console-only prompt
	// flushing, which is not guaranteed when the safe pipe backend is active.
	deadline := time.Now().Add(10 * time.Second)
	buf := make([]byte, 4096)
	var out strings.Builder
	for time.Now().Before(deadline) {
		n, err := backend.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
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
