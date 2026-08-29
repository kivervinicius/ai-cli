package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kivervinicius/ai-cli/internal/app"
	"github.com/kivervinicius/ai-cli/internal/browser"
)

func main() {
	base := filepath.Base(os.Args[0])
	if base == "nexus-browser" || base == "ai-browser" || base == "xdg-open" {
		if err := browser.Open(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Nexus browser helper:", err)
			os.Exit(1)
		}
		return
	}

	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", base, err)
		os.Exit(1)
	}
}
