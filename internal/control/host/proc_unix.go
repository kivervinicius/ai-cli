//go:build !windows

package host

import "os/exec"

// prepareCmd prepares command execution.
// PTY backends handle session/process group creation via Setsid.
func prepareCmd(cmd *exec.Cmd) {}
