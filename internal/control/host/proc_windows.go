//go:build windows

package host

import (
	"os"
	"os/exec"
)

func prepareCmd(cmd *exec.Cmd) {}

func signalProcessGroup(proc *os.Process, sig os.Signal) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(sig)
}

func killProcessGroup(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Kill()
}
