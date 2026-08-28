// Package buildinfo is the single source of truth for binary version metadata.
//
// Values are injected at build time via -ldflags:
//
//	-X github.com/kivervinicius/ai-cli/internal/buildinfo.Version=1.2.3
//	-X github.com/kivervinicius/ai-cli/internal/buildinfo.Commit=abc1234
//	-X github.com/kivervinicius/ai-cli/internal/buildinfo.BuildDate=2026-08-28T00:00:00Z
//
// When built without ldflags (e.g. `go run`), values fall back to dev/unknown.
package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the binary.
	Version = "dev"
	// Commit is the git commit the binary was built from.
	Commit = "unknown"
	// BuildDate is the UTC build timestamp.
	BuildDate = "unknown"
)

// Go returns the Go toolchain version.
func Go() string { return runtime.Version() }

// Platform returns "os/arch".
func Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// String renders a single-line human-readable version string.
func String() string {
	return fmt.Sprintf("ai-cli %s (%s, %s) commit %s built %s",
		Version, Platform(), Go(), Commit, BuildDate)
}

// JSON returns the version metadata as a JSON-serializable map.
func JSON() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     Commit,
		"build_date": BuildDate,
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"go":         Go(),
	}
}
