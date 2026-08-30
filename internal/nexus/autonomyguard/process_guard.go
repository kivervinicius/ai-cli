package autonomyguard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Policy is the process-level subset of the Mission AutonomyContract.
type Policy struct {
	DisallowDestructiveGit bool
	AllowGitPush           bool
	AllowDeploy            bool
	AllowExternalNetwork   bool
	AllowSecretAccess      bool
	AllowPaidServices      bool
}

var externalNetworkTools = map[string]struct{}{
	"curl": {}, "wget": {}, "ssh": {}, "scp": {}, "sftp": {}, "ftp": {}, "nc": {}, "ncat": {}, "telnet": {},
}

var secretAccessTools = map[string]struct{}{
	"vault": {}, "op": {}, "pass": {}, "secret-tool": {},
}

var paidServiceTools = map[string]struct{}{
	"aws": {}, "gcloud": {}, "az": {}, "doctl": {}, "fly": {}, "vercel": {}, "netlify": {}, "pulumi": {},
}

var deployMutations = map[string][]string{
	"kubectl":   {"apply", "delete", "create", "patch", "replace", "edit", "scale", "rollout", "set", "label", "annotate", "taint", "cordon", "uncordon", "drain"},
	"helm":      {"install", "upgrade", "uninstall", "rollback"},
	"terraform": {"apply", "destroy", "import", "taint", "untaint"},
	"docker":    {"push", "login", "logout", "swarm", "stack"},
	"npm":       {"publish", "unpublish", "deprecate", "dist-tag", "owner", "token", "access"},
	"yarn":      {"npm publish", "npm tag", "npm audit --fix"},
	"pnpm":      {"publish", "deploy"},
	"bun":       {"publish"},
	"gh":        {"api", "release", "workflow run", "pr create", "pr merge", "repo create", "repo delete", "issue create", "secret set", "variable set"},
}

// WriteCommandGuards creates PATH shims that proxy safe commands to their real
// binaries while rejecting high-impact local/remote mutations prohibited by the
// AutonomyContract. Missing optional CLIs are ignored.
func WriteCommandGuards(dir string, policy Policy) ([]string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	var tools []string
	if policy.DisallowDestructiveGit || !policy.AllowGitPush {
		tools = append(tools, "git")
	}
	if !policy.AllowDeploy {
		for tool := range deployMutations {
			tools = append(tools, tool)
		}
	}
	if !policy.AllowExternalNetwork {
		for tool := range externalNetworkTools {
			tools = append(tools, tool)
		}
	}
	if !policy.AllowSecretAccess {
		for tool := range secretAccessTools {
			tools = append(tools, tool)
		}
	}
	if !policy.AllowPaidServices {
		for tool := range paidServiceTools {
			tools = append(tools, tool)
		}
	}
	tools = uniqueToolNames(tools)
	sort.Strings(tools)
	created := make([]string, 0, len(tools))
	for _, tool := range tools {
		real, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		// Avoid recursively wrapping an already prepared guard directory.
		if filepath.Clean(filepath.Dir(real)) == filepath.Clean(dir) {
			continue
		}
		name := tool
		content := ""
		mode := os.FileMode(0700)
		if runtime.GOOS == "windows" {
			name += ".cmd"
			content = renderWindowsCommandGuard(real, tool, policy)
			mode = 0600
		} else {
			content = renderUnixCommandGuard(real, tool, policy)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			return nil, err
		}
		created = append(created, path)
	}
	return created, nil
}

func uniqueToolNames(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, tool := range in {
		if tool != "" && !seen[tool] {
			seen[tool] = true
			out = append(out, tool)
		}
	}
	return out
}

func blockEntireTool(tool string, policy Policy) bool {
	if _, ok := externalNetworkTools[tool]; ok && !policy.AllowExternalNetwork {
		return true
	}
	if _, ok := secretAccessTools[tool]; ok && !policy.AllowSecretAccess {
		return true
	}
	if _, ok := paidServiceTools[tool]; ok && !policy.AllowPaidServices {
		return true
	}
	return false
}

func blockedPatterns(tool string, policy Policy) []string {
	var patterns []string
	if tool == "git" {
		if !policy.AllowGitPush {
			patterns = append(patterns, "push")
		}
		if policy.DisallowDestructiveGit {
			patterns = append(patterns, "reset --hard", "clean -f", "clean -d", "branch -D", "checkout --", "restore ")
		}
		return patterns
	}
	if !policy.AllowDeploy {
		return append(patterns, deployMutations[tool]...)
	}
	return nil
}

func renderUnixCommandGuard(real, tool string, policy Policy) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\nARGS=\" $* \"\nblock() { echo \"NEXUS_AUTONOMY_BLOCKED: ")
	b.WriteString(tool)
	b.WriteString(" $*\" >&2; exit 126; }\n")
	if blockEntireTool(tool, policy) {
		b.WriteString("block\n")
	}
	for _, pattern := range blockedPatterns(tool, policy) {
		words := strings.Fields(pattern)
		if len(words) == 1 {
			fmt.Fprintf(&b, "case \"$ARGS\" in *\" %s \"*) block ;; esac\n", words[0])
			continue
		}
		// Requiring each token catches common flag combinations while remaining
		// shell-portable. False positives err on the side of the autonomy boundary.
		fmt.Fprintf(&b, "case \"$ARGS\" in *\" %s \"*\" %s \"*) block ;; esac\n", words[0], words[1])
	}
	fmt.Fprintf(&b, "exec %s \"$@\"\n", shellQuote(real))
	return b.String()
}

func renderWindowsCommandGuard(real, tool string, policy Policy) string {
	var b strings.Builder
	b.WriteString("@echo off\r\nsetlocal\r\nset \"ARGS= %* \"\r\n")
	if blockEntireTool(tool, policy) {
		b.WriteString("goto :blocked\r\n")
	}
	for _, pattern := range blockedPatterns(tool, policy) {
		words := strings.Fields(pattern)
		if len(words) == 1 {
			fmt.Fprintf(&b, "echo %%ARGS%% | findstr /I /C:\" %s \" >nul && goto :blocked\r\n", words[0])
		} else {
			fmt.Fprintf(&b, "echo %%ARGS%% | findstr /I /C:\" %s \" >nul && echo %%ARGS%% | findstr /I /C:\" %s \" >nul && goto :blocked\r\n", words[0], words[1])
		}
	}
	fmt.Fprintf(&b, "\"%s\" %%*\r\nexit /b %%ERRORLEVEL%%\r\n:blocked\r\necho NEXUS_AUTONOMY_BLOCKED: %s %%* 1>&2\r\nexit /b 126\r\n", strings.ReplaceAll(real, "\"", "\"\""), tool)
	return b.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
