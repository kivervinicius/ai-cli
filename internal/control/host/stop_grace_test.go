package host

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestGracefulStopForProjectShellIsShort(t *testing.T) {
	if gracefulStopWait("shell") > 300*time.Millisecond {
		t.Fatalf("shell stop wait too long: %s", gracefulStopWait("shell"))
	}
	if gracefulStopSignal("shell") != syscall.SIGTERM {
		t.Fatalf("shell stop should use SIGTERM, got %v", gracefulStopSignal("shell"))
	}
	if gracefulStopWait("codex") != 3*time.Second {
		t.Fatalf("agent stop wait should stay 3s, got %s", gracefulStopWait("codex"))
	}
	if gracefulStopSignal("codex") != os.Interrupt {
		t.Fatalf("agent stop should use SIGINT, got %v", gracefulStopSignal("codex"))
	}
}
