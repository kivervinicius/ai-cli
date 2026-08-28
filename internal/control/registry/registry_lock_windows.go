//go:build windows

package registry

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK = 0x00000002
)

// acquireFileLock obtains a cross-process lock on the registry file to prevent race conditions.
func acquireFileLock(path string) (func() error, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	var overlapped syscall.Overlapped
	// Block until we can acquire the lock
	ret, _, errCode := procLockFileEx.Call(
		f.Fd(),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK),
		0,
		0xFFFFFFFF,
		0xFFFFFFFF,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		f.Close()
		return nil, fmt.Errorf("LockFileEx failed: %v", errCode)
	}

	return func() error {
		defer f.Close()
		var ov syscall.Overlapped
		ret, _, errCode := procUnlockFileEx.Call(
			f.Fd(),
			0,
			0xFFFFFFFF,
			0xFFFFFFFF,
			uintptr(unsafe.Pointer(&ov)),
		)
		if ret == 0 {
			return fmt.Errorf("UnlockFileEx failed: %v", errCode)
		}
		return nil
	}, nil
}
