//go:build windows

package terminal

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestUTF16EnvBlockIsDoubleNullTerminated(t *testing.T) {
	block := utf16EnvBlock([]string{"NEXUS_A=one", "NEXUS_B=two"})
	decoded := string(utf16.Decode(block))
	if !strings.HasSuffix(decoded, "\x00\x00") {
		t.Fatalf("environment block must end with two NULs: %q", decoded)
	}
	if strings.Contains(decoded[:len(decoded)-2], "\x00\x00") {
		t.Fatalf("environment block contains an early double NUL: %q", decoded)
	}
	if !strings.Contains(decoded, "NEXUS_A=one\x00NEXUS_B=two\x00") {
		t.Fatalf("environment entries were not encoded contiguously: %q", decoded)
	}
}
