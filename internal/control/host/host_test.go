package host

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestSlashRouterInterceptionAndEscape(t *testing.T) {
	sess := registry.RuntimeSession{
		RuntimeID:         "rt-test-01",
		ProviderID:        "codex",
		ProfileID:         "work",
		ProviderSessionID: "sess-abc-123",
		Workspace:         "/workspace/project",
		State:             registry.StateRunning,
		ControlLevel:      registry.ControlLevelAPI,
	}

	// 1. Normal prompt (should forward untouched)
	res1 := RouteSlashCommand("Refactor this file\n", sess)
	if res1.Intercepted || res1.ForwardToProcess != "Refactor this file\n" {
		t.Errorf("expected normal input to be forwarded, got %+v", res1)
	}

	// 2. /ai status (should be intercepted)
	res2 := RouteSlashCommand("/ai status\n", sess)
	if !res2.Intercepted || res2.ForwardToProcess != "" || !strings.Contains(res2.Response, "rt-test-01") {
		t.Errorf("expected /ai status to be intercepted, got %+v", res2)
	}

	// 3. /ai help (should be intercepted)
	res3 := RouteSlashCommand("/ai help\n", sess)
	if !res3.Intercepted || !strings.Contains(res3.Response, "UNIVERSAL SLASH COMMANDS") {
		t.Errorf("expected /ai help to show guide, got %+v", res3)
	}

	// 4. //ai escaped command (should unescape and forward to provider)
	res4 := RouteSlashCommand("//ai tell me a story\n", sess)
	if res4.Intercepted || res4.ForwardToProcess != "/ai tell me a story\n" {
		t.Errorf("expected //ai to be unescaped to /ai, got %+v", res4)
	}

	// 5. /ai detach
	res5 := RouteSlashCommand("/ai detach", sess)
	if !res5.Intercepted || res5.Action != "detach" {
		t.Errorf("expected /ai detach to produce action detach, got %+v", res5)
	}

	// 6. /ai stop
	res6 := RouteSlashCommand("/ai stop", sess)
	if !res6.Intercepted || res6.Action != "stop" {
		t.Errorf("expected /ai stop to produce action stop, got %+v", res6)
	}
}

func TestRingBufferBounding(t *testing.T) {
	rb := NewRingBuffer(10)
	_, _ = rb.Write([]byte("12345"))
	if string(rb.Bytes()) != "12345" {
		t.Errorf("expected '12345', got %q", string(rb.Bytes()))
	}

	// Overflow buffer
	_, _ = rb.Write([]byte("67890ABCDE"))
	if string(rb.Bytes()) != "67890ABCDE" {
		t.Errorf("expected '67890ABCDE', got %q", string(rb.Bytes()))
	}
}

func TestSessionHostLifecycle(t *testing.T) {
	runtimeID := "rt-host-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  "cat",
		Args:    []string{},
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	time.Sleep(100 * time.Millisecond)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect to SessionHost: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	st, err := client.Status()
	if err != nil {
		t.Fatalf("status query failed: %v", err)
	}
	if st.RuntimeID != runtimeID || st.State != "RUNNING" {
		t.Errorf("unexpected status: %+v", st)
	}

	// Send Stop command
	if err := client.Stop(); err != nil {
		t.Errorf("stop request failed: %v", err)
	}
}

func TestSessionHost_CmdInputNoDeadlock(t *testing.T) {
	runtimeID := "rt-deadlock-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  "cat",
		Args:    []string{},
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	time.Sleep(100 * time.Millisecond)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect to SessionHost: %v", err)
	}
	defer client.Close()

	// Send CmdInput via RPC with a strict timeout
	done := make(chan error, 1)
	go func() {
		_, err := client.Send(protocol.CmdInput, protocol.InputPayload{Data: "hello\n"})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CmdInput failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: CmdInput hung for >2s due to reentrant mutex lock")
	}
}

func TestSessionHost_SlowObserverDoesNotBlockWriter(t *testing.T) {
	runtimeID := "rt-fanout-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  "cat",
		Args:    []string{},
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	time.Sleep(100 * time.Millisecond)

	// Writer client
	writer, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect writer: %v", err)
	}
	defer writer.Close()
	_, _ = writer.Send(protocol.CmdAttach, nil)

	// Observer client that never reads
	observer, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect observer: %v", err)
	}
	defer observer.Close()
	_, _ = observer.Send(protocol.CmdAttach, nil)

	// Send large amount of input through writer
	writeDone := make(chan error, 1)
	go func() {
		for i := 0; i < 50; i++ {
			_, err := writer.RawConn().Write([]byte("echo test line\n"))
			if err != nil {
				writeDone <- err
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		writeDone <- nil
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("writer failed while observer is slow: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("BLOCKED: writer was blocked by slow observer")
	}
}


