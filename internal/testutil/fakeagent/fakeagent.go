package fakeagent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Run simulates a deterministic AI agent CLI for integration and E2E tests.
func Run(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	fmt.Fprintf(out, "FakeAgent v1.0.0 ready (Session: 0195fake-session-id)\n")

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case "exit", "quit":
			fmt.Fprintf(out, "FakeAgent exiting...\n")
			return

		case "rate-limit":
			fmt.Fprintf(out, "ERROR: HTTP 429 Too Many Requests (Quota Exceeded). Resets in 45m\n")

		case "crash":
			os.Exit(2)

		case "sleep":
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintf(out, "FakeAgent awake\n")

		default:
			fmt.Fprintf(out, "FakeAgent Echo: %s\n", line)
		}
	}
}
