//go:build windows

package host

var testTerminalBinary = "cmd.exe"

func testTerminalArgs() []string {
	// Keep the child alive for lifecycle/IPC tests. `more` exits immediately
	// when its stdin is not a console, making the named pipe disappear before
	// the test client can connect.
	return []string{"/D", "/Q", "/C", "ping -t 127.0.0.1 >nul"}
}
