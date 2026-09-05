package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalWorkspacePath resolves and canonicalizes any workspace or project path:
// - Verifies non-empty
// - Computes absolute path
// - Cleans path separators and relative segments
// - Resolves filesystem symlinks (e.g. macOS /var -> /private/var)
// - Preserves clean canonical path
func CanonicalWorkspacePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errors.New("workspace path cannot be empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	clean := filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		clean = filepath.Clean(resolved)
	}
	return clean, nil
}

// CanonicalExistingWorkspaceDir verifies that the canonical path exists and is a directory.
func CanonicalExistingWorkspaceDir(p string) (string, error) {
	clean, err := CanonicalWorkspacePath(p)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("workspace path does not exist: %w", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("workspace path is not a directory: %s", clean)
	}
	return clean, nil
}
