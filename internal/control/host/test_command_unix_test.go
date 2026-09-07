//go:build !windows

package host

var testTerminalBinary = "cat"

func testTerminalArgs() []string { return nil }
