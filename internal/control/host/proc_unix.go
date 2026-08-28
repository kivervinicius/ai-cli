//go:build !windows

package host

import (
	"os"
	"os/exec"
	"syscall"
)

// prepareCmd prepares command execution.
// PTY backends handle session/process group creation via Setsid.
// Non-PTY backends configure Setpgid in the terminal fallback.
func prepareCmd(cmd *exec.Cmd) {}

// signalProcessGroup sends a signal to the process group of the process.
func signalProcessGroup(proc *os.Process, sig os.Signal) error {
	if proc == nil || proc.Pid <= 0 {
		return nil
	}
	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		sysSig = syscall.SIGTERM
	}
	// Kill the process group using negative PID (-pid)
	err := syscall.Kill(-proc.Pid, sysSig)
	if err != nil {
		return proc.Signal(sig)
	}
	return nil
}

// killProcessGroup forcefully kills the process group of the process.
func killProcessGroup(proc *os.Process) error {
	if proc == nil || proc.Pid <= 0 {
		return nil
	}
	err := syscall.Kill(-proc.Pid, syscall.SIGKILL)
	if err != nil {
		return proc.Kill()
	}
	return nil
}
