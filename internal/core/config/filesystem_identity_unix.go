//go:build !windows

package config

import (
	"fmt"
	"os"
	"syscall"
)

func filesystemIdentity(path string) (FilesystemIdentity, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FilesystemIdentity{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return FilesystemIdentity{Kind: "textual", StableKey: path, Available: false}, nil
	}
	return FilesystemIdentity{
		Kind:      "unix-device-inode",
		StableKey: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino),
		Available: true,
	}, nil
}
