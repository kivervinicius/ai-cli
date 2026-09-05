package runtime

import (
	"runtime"
	"testing"
)

func TestDefaultCredentialIsolatorPlatform(t *testing.T) {
	iso := DefaultCredentialIsolator()
	if iso == nil {
		t.Fatal("expected non-nil isolator")
	}
	if iso.Platform() != runtime.GOOS {
		t.Fatalf("expected platform %q, got %q", runtime.GOOS, iso.Platform())
	}
}

func TestCredentialIsolationWrapCommand(t *testing.T) {
	iso := DefaultCredentialIsolator()
	bin, args := iso.WrapCommand("echo", []string{"hi"})
	if bin == "" {
		t.Fatal("expected non-empty bin")
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if bin != "echo" || len(args) != 1 || args[0] != "hi" {
			t.Fatalf("expected passthrough on %s, got bin=%q, args=%v", runtime.GOOS, bin, args)
		}
	}
}
