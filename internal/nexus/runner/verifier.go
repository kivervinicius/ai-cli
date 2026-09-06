package runner

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// VerificationEngine executes concrete gates against the package workspace.
type VerificationEngine struct{}

func NewVerificationEngine() *VerificationEngine { return &VerificationEngine{} }

// RunVerification executes each approved command through the platform shell so
// quoting and package-manager syntax behave consistently on Unix and Windows.
func (v *VerificationEngine) RunVerification(ctx context.Context, workspace string, commands []string) []VerificationResult {
	var results []VerificationResult
	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}
		start := time.Now()
		cmd := platformShellCommand(ctx, cmdStr)
		cmd.Dir = workspace
		out, err := cmd.CombinedOutput()
		dur := time.Since(start).Milliseconds()
		exitCode, passed := 0, err == nil
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		snippet := string(out)
		if len(snippet) > 8192 {
			snippet = snippet[len(snippet)-8192:]
		}
		results = append(results, VerificationResult{Command: cmdStr, Passed: passed, ExitCode: exitCode, OutputSnippet: snippet, DurationMs: dur, VerifiedAt: time.Now().UTC()})
		if !passed {
			break
		}
	}
	return results
}

func platformShellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", command)
	}
	return exec.CommandContext(ctx, "/bin/sh", "-c", command)
}
