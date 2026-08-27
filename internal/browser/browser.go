package browser

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open opens a URL using host browser utilities safely.
func Open(args []string) error {
	if len(args) == 0 {
		return nil
	}
	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL %q", url)
	}

	// Try standard browser commands
	for _, cmd := range []string{"xdg-open", "google-chrome", "firefox", "chromium", "brave"} {
		if path, err := exec.LookPath(cmd); err == nil {
			c := exec.Command(path, url)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Start()
		}
	}

	fmt.Printf("\nPlease open the following authentication URL in your browser:\n\n  %s\n\n", url)
	return nil
}
