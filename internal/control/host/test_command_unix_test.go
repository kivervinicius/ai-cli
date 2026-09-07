//go:build !windows

package host

var testTerminalBinary = "cat"

func testTerminalArgs() []string { return nil }

var interactiveTestTerminalBinary = "sh"

func interactiveTestTerminalArgs() []string {
	return []string{"-c", "printf 'NEXUS_TEST_READY\\n'; exec cat"}
}
