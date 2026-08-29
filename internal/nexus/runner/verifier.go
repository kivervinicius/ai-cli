package runner

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// VerificationEngine executes concrete test/lint commands against the workspace (§Phase H).
type VerificationEngine struct{}

func NewVerificationEngine() *VerificationEngine {
	return &VerificationEngine{}
}

// RunVerification executes commands in the given workspace directory and returns structured evidence.
func (v *VerificationEngine) RunVerification(ctx context.Context, workspace string, commands []string) []VerificationResult {
	var results []VerificationResult

	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}

		start := time.Now()
		parts := strings.Fields(cmdStr)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		cmd.Dir = workspace

		out, err := cmd.CombinedOutput()
		dur := time.Since(start).Milliseconds()

		exitCode := 0
		passed := true
		if err != nil {
			passed = false
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		snippet := string(out)
		if len(snippet) > 2048 {
			snippet = snippet[len(snippet)-2048:] // Keep last 2KB
		}

		results = append(results, VerificationResult{
			Command:       cmdStr,
			Passed:        passed,
			ExitCode:      exitCode,
			OutputSnippet: snippet,
			DurationMs:    dur,
			VerifiedAt:    time.Now().UTC(),
		})

		// Stop at first failure
		if !passed {
			break
		}
	}

	return results
}
