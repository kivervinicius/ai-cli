//go:build !windows

package registry

import (
	"fmt"
	"os"
	"syscall"
)

// acquireFileLock obtains a cross-process lock on the registry file to prevent race conditions.
func acquireFileLock(path string) (func() error, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Block until we can acquire the lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to lock registry: %w", err)
	}

	return func() error {
		defer f.Close()
		return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}, nil
}
