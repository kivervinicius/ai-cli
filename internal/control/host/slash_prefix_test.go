package host

import (
	"testing"
)

func TestSlashPrefixRouter_NeverLeaksToChild(t *testing.T) {
	router := NewSlashPrefixRouter()

	// 1. Send "/ai status\r"
	input := []byte("/ai status\r")
	var forwarded []byte
	var interceptedCmd string

	for _, b := range input {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if len(forwarded) != 0 {
		t.Errorf("CRITICAL LEAK: /ai status leaked bytes to child: %q", string(forwarded))
	}
	if interceptedCmd != "/ai status" {
		t.Errorf("expected intercepted command '/ai status', got %q", interceptedCmd)
	}

	// 2. Normal prompt: "hello world\r" (should forward completely)
	router.Reset()
	forwarded = nil
	interceptedCmd = ""
	for _, b := range []byte("hello world\r") {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if string(forwarded) != "hello world\r" {
		t.Errorf("expected 'hello world\\r', got %q", string(forwarded))
	}
	if interceptedCmd != "" {
		t.Errorf("unexpected command interception: %q", interceptedCmd)
	}

	// 3. Provider native slash command: "/help\r" (should forward "/help\r" untouched)
	router.Reset()
	forwarded = nil
	interceptedCmd = ""
	for _, b := range []byte("/help\r") {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if string(forwarded) != "/help\r" {
		t.Errorf("expected '/help\\r', got %q", string(forwarded))
	}
	if interceptedCmd != "" {
		t.Errorf("unexpected command interception: %q", interceptedCmd)
	}

	// 4. Escaped command: "//ai prompt\r" (should forward "/ai prompt\r")
	router.Reset()
	forwarded = nil
	interceptedCmd = ""
	for _, b := range []byte("//ai prompt\r") {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if string(forwarded) != "/ai prompt\r" {
		t.Errorf("expected unescaped '/ai prompt\\r', got %q", string(forwarded))
	}
	if interceptedCmd != "" {
		t.Errorf("unexpected command interception: %q", interceptedCmd)
	}

	// 5. Send "/nexus status\r"
	router.Reset()
	forwarded = nil
	interceptedCmd = ""
	for _, b := range []byte("/nexus status\r") {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if len(forwarded) != 0 {
		t.Errorf("CRITICAL LEAK: /nexus status leaked bytes to child: %q", string(forwarded))
	}
	if interceptedCmd != "/nexus status" {
		t.Errorf("expected intercepted command '/nexus status', got %q", interceptedCmd)
	}

	// 6. Escaped command: "//nexus prompt\r" (should forward "/nexus prompt\r")
	router.Reset()
	forwarded = nil
	interceptedCmd = ""
	for _, b := range []byte("//nexus prompt\r") {
		out := router.ProcessByte(b)
		if out.Action == ActionForwardBytes {
			forwarded = append(forwarded, out.ForwardBytes...)
		} else if out.Action == ActionControlCommand {
			interceptedCmd = out.ControlCmd
		}
	}

	if string(forwarded) != "/nexus prompt\r" {
		t.Errorf("expected unescaped '/nexus prompt\\r', got %q", string(forwarded))
	}
	if interceptedCmd != "" {
		t.Errorf("unexpected command interception: %q", interceptedCmd)
	}
}
