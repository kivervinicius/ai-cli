//go:build windows

package registry

import (
	"syscall"
	"unsafe"
)

var (
	modKernel32Process     = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = modKernel32Process.NewProc("OpenProcess")
	procGetExitCodeProcess = modKernel32Process.NewProc("GetExitCodeProcess")
	procGetProcessTimes    = modKernel32Process.NewProc("GetProcessTimes")
	procCloseHandle        = modKernel32Process.NewProc("CloseHandle")
)

const (
	processQueryLimitedInfo = 0x1000
	stillActive             = 259
)

// IsProcessAlive checks whether a process with the given PID is currently running on Windows.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, _, _ := procOpenProcess.Call(uintptr(processQueryLimitedInfo), 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)

	var exitCode uint32
	ret, _, _ := procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return false
	}
	return exitCode == stillActive
}

// IsProcessAliveWithGeneration checks whether a process is alive and its creation time
// roughly matches the recorded hostGeneration (to detect PID recycling).
func IsProcessAliveWithGeneration(pid int, hostGeneration int64) bool {
	if !IsProcessAlive(pid) {
		return false
	}
	if hostGeneration <= 0 {
		return true
	}

	h, _, _ := procOpenProcess.Call(uintptr(processQueryLimitedInfo), 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	ret, _, _ := procGetProcessTimes.Call(
		h,
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret != 0 {
		creationNano := creationTime.Nanoseconds()
		// If process was created more than 10 seconds after our recorded hostGeneration,
		// it is very likely a recycled PID.
		if creationNano > hostGeneration+(10*1e9) {
			return false
		}
	}
	return true
}
