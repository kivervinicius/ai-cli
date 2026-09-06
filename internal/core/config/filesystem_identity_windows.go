//go:build windows

package config

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// filesystemIdentity opens an existing directory with backup semantics and
// asks Windows for its final volume path. This resolves drive-letter aliases,
// 8.3 short names, junctions and the extended-length prefix through the same
// namespace used by the filesystem.
func filesystemIdentity(path string) (FilesystemIdentity, error) {
	if _, err := os.Stat(path); err != nil {
		return FilesystemIdentity{}, err
	}

	nativePath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return FilesystemIdentity{}, fmt.Errorf("encode filesystem path: %w", err)
	}
	handle, err := windows.CreateFile(
		nativePath,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return FilesystemIdentity{}, fmt.Errorf("open filesystem path: %w", err)
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, 256)
	for {
		n, callErr := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
		if callErr == nil && n < uint32(len(buffer)) {
			finalPath := string(utf16.Decode(buffer[:n]))
			finalPath = strings.TrimPrefix(finalPath, `\\?\`)
			return FilesystemIdentity{Kind: "windows-final-path", StableKey: finalPath, Available: true}, nil
		}
		if callErr != nil && callErr != windows.ERROR_INSUFFICIENT_BUFFER {
			return FilesystemIdentity{}, fmt.Errorf("resolve final filesystem path: %w", callErr)
		}
		if n == 0 || n >= 32768 {
			return FilesystemIdentity{}, fmt.Errorf("resolve final filesystem path: invalid length %d", n)
		}
		buffer = make([]uint16, n+1)
	}
}
