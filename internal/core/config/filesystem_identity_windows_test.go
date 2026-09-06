//go:build windows

package config

import (
	"os"
	"testing"
)

func TestFilesystemIdentityForExistingDirectoryUsesNativeIdentity(t *testing.T) {
	dir := t.TempDir()
	identity, err := filesystemIdentity(dir)
	if err != nil {
		t.Fatalf("filesystemIdentity(%q): %v", dir, err)
	}
	if !identity.Available || identity.Kind != "windows-final-path" || identity.StableKey == "" {
		t.Fatalf("expected native identity for existing directory, got %+v", identity)
	}
	if _, err := os.Stat(identity.StableKey); err != nil {
		t.Fatalf("native identity should resolve to an existing path, got %q: %v", identity.StableKey, err)
	}
}
