package launchenv

import (
	"os"
	"strings"
)

// Isolation keys must stay under driver control. Agent Environment must never
// point Codex/Claude/etc. back at the host home (broken plugins, credential bleed).
var protectedIsolationKeys = map[string]struct{}{
	"HOME":             {},
	"CODEX_HOME":       {},
	"CODEX_CONFIG_DIR": {},
	"CURSOR_HOME":      {},
	"GEMINI_CLI_HOME":  {},
	"OPENCODE_HOME":    {},
	"XDG_DATA_HOME":    {},
	"XDG_CONFIG_HOME":  {},
	"XDG_STATE_HOME":   {},
	"XDG_CACHE_HOME":   {},
}

// Merge canonicalizes environment overrides and optionally prepends directories
// to the PATH already produced by a provider driver. This avoids duplicate keys
// whose precedence differs across operating systems/process launchers.
func Merge(base []string, overrides map[string]string, pathPrepend []string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	seen := map[string]bool{}
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		key := parts[0]
		if !seen[key] {
			order = append(order, key)
			seen[key] = true
		}
		values[key] = parts[1]
	}
	for key, value := range overrides {
		if key == "" {
			continue
		}
		if _, protected := protectedIsolationKeys[key]; protected {
			continue
		}
		if !seen[key] {
			order = append(order, key)
			seen[key] = true
		}
		values[key] = value
	}
	if len(pathPrepend) > 0 {
		parts := make([]string, 0, len(pathPrepend)+1)
		for _, dir := range pathPrepend {
			if strings.TrimSpace(dir) != "" {
				parts = append(parts, dir)
			}
		}
		if existing := values["PATH"]; existing != "" {
			parts = append(parts, existing)
		}
		values["PATH"] = strings.Join(parts, string(os.PathListSeparator))
		if !seen["PATH"] {
			order = append(order, "PATH")
			seen["PATH"] = true
		}
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+values[key])
	}
	return out
}
