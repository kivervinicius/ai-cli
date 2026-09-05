//go:build windows

package config

import "os"

// Native volume/file-ID lookup is kept behind this OS boundary. Until the
// handle-based implementation is available, Windows reports a truthful
// textual fallback instead of pretending that path text is a stable identity.
func filesystemIdentity(path string) (FilesystemIdentity, error) {
	if _, err := os.Stat(path); err != nil {
		return FilesystemIdentity{}, err
	}
	return FilesystemIdentity{Kind: "textual", StableKey: path, Available: false}, nil
}
