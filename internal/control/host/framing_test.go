package host

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadBoundedLine(t *testing.T) {
	// Normal newline-terminated frame.
	r := bufio.NewReader(strings.NewReader(`{"command":"ping"}` + "\n"))
	line, err := readBoundedLine(r, 64)
	if err != nil || string(line) != `{"command":"ping"}` {
		t.Fatalf("unexpected line=%q err=%v", line, err)
	}

	// Empty frame.
	r = bufio.NewReader(strings.NewReader("\n"))
	line, err = readBoundedLine(r, 64)
	if err != nil || len(line) != 0 {
		t.Fatalf("expected empty line, got %q err=%v", line, err)
	}

	// Oversized frame without newline must error before unbounded growth.
	big := strings.Repeat("a", 128)
	r = bufio.NewReader(strings.NewReader(big))
	_, err = readBoundedLine(r, 64)
	if err != errFrameTooLarge {
		t.Fatalf("expected errFrameTooLarge, got %v", err)
	}

	// EOF mid-frame.
	r = bufio.NewReader(strings.NewReader("partial"))
	_, err = readBoundedLine(r, 64)
	if err == nil {
		t.Fatal("expected EOF error")
	}

	// Exactly at limit is allowed.
	exact := strings.Repeat("b", 64) + "\n"
	r = bufio.NewReader(strings.NewReader(exact))
	line, err = readBoundedLine(r, 64)
	if err != nil || len(line) != 64 {
		t.Fatalf("expected 64-byte frame without error, got %d err=%v", len(line), err)
	}
}

func FuzzReadBoundedLine(f *testing.F) {
	f.Add(strings.Repeat("x", 10) + "\n")
	f.Add(strings.Repeat("x", 200))
	f.Add("")
	for i := 0; i < 16; i++ {
		f.Add(strings.Repeat("y", i*10) + "\n")
	}
	f.Fuzz(func(t *testing.T, input string) {
		r := bufio.NewReader(strings.NewReader(input))
		line, err := readBoundedLine(r, 64)
		if err == errFrameTooLarge && len(line) <= 64 {
			t.Fatalf("oversized frame reported with %d bytes", len(line))
		}
		if err == nil && len(line) > 64 {
			t.Fatalf("bounded reader returned %d bytes over limit", len(line))
		}
	})
}
