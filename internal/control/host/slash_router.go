package host

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// SlashResult represents the outcome of processing an input line through the Slash Router.
type SlashResult struct {
	Intercepted      bool   // True if command was an /ai slash command and consumed by AI Control
	Response         string // Formatted response to display to the user
	ForwardToProcess string // Text to forward to child process (if any, e.g. unescaped //ai text)
	Action           string // Special action e.g. "detach", "stop", "handoff", "continue"
	ActionArg        string // Argument for special action (e.g. target profile)
}

// StripANSI removes ANSI escape sequences and non-printable control codes from input strings.
func StripANSI(str string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c == 0x1b { // ESC
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '~' {
				inEsc = false
			}
			continue
		}
		if c >= 32 || c == '\t' {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// RouteSlashCommand inspects terminal input lines and intercepts /nexus or /ai commands with live usage data.
func RouteSlashCommand(input string, session registry.RuntimeSession) SlashResult {
	clean := StripANSI(input)
	trimmed := strings.TrimSpace(clean)

	// 1. Check for escape prefix "//nexus" or "//ai"
	if strings.HasPrefix(trimmed, "//nexus") {
		escaped := "/nexus" + trimmed[7:]
		if strings.HasSuffix(input, "\n") {
			escaped += "\n"
		}
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: escaped,
		}
	}
	if strings.HasPrefix(trimmed, "//ai") {
		escaped := "/ai" + trimmed[4:]
		if strings.HasSuffix(input, "\n") {
			escaped += "\n"
		}
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: escaped,
		}
	}

	// 2. Check for reserved "/nexus" or "/ai" command
	isNexus := strings.HasPrefix(trimmed, "/nexus")
	isAI := strings.HasPrefix(trimmed, "/ai")
	if !isNexus && !isAI {
		// Normal input: forward untouched
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: input,
		}
	}

	prefix := "/nexus"
	if isAI && !isNexus {
		prefix = "/ai"
	}

	// It is a slash command: intercept completely
	parts := strings.Fields(trimmed)
	subcmd := ""
	if len(parts) > 1 {
		subcmd = parts[1]
	}

	switch subcmd {
	case "help", "?", "":
		resp := fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════╗
║             NEXUS CONTROL — UNIVERSAL SLASH COMMANDS            ║
╠══════════════════════════════════════════════════════════════════╣
║  %s status               Show current runtime status & quota   ║
║  %s accounts             List available accounts & quotas      ║
║  %s usage                Display honest usage metrics snapshot ║
║  %s handoff <profile>    Same-provider account handoff         ║
║  %s continue <provider>  Cross-provider context handoff        ║
║  %s control              Open Control Center TUI instructions  ║
║  %s detach               Detach from session (keeps alive)     ║
║  %s stop                 Gracefully terminate runtime session  ║
║  /%s <text>              Escape prefix to send literal command ║
╚══════════════════════════════════════════════════════════════════╝
`, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
		return SlashResult{Intercepted: true, Response: resp}

	case "status":
		snap := profile.GetUsageSnapshot(session.ProviderID, session.ProfileID)
		usageStr := formatUsageSummary(snap)

		resp := fmt.Sprintf(`
┌─ NEXUS CONTROL STATUS ───────────────────────────────────────┐
│  Runtime ID:       %-41s │
│  Provider:         %-41s │
│  Profile:          %-41s │
│  Provider Session: %-41s │
│  State:            %-41s │
│  Control Level:    %-41s │
│  Workspace:        %-41s │
│  Usage / Quota:    %-41s │
└──────────────────────────────────────────────────────────────┘
`, session.RuntimeID, strings.ToUpper(session.ProviderID), session.ProfileID,
			fallbackUnknown(session.ProviderSessionID), session.State, session.ControlLevel,
			truncateStr(session.Workspace, 41), usageStr)
		return SlashResult{Intercepted: true, Response: resp}

	case "accounts":
		profs, _ := profile.List()
		var lines []string
		lines = append(lines, fmt.Sprintf("=== Accounts & Quotas for Provider: %s ===", strings.ToUpper(session.ProviderID)))
		lines = append(lines, fmt.Sprintf("%-16s %-12s %-14s %-14s %s", "PROFILE", "STATUS", "CAPACITY", "FRESHNESS", "RESET"))
		lines = append(lines, strings.Repeat("─", 70))

		found := 0
		for _, p := range profs {
			if p.Provider == session.ProviderID {
				found++
				activeMarker := "  "
				if p.Name == session.ProfileID {
					activeMarker = "> "
				}
				info := profile.GetAccountInfo(p.Provider, p.Name)
				snap := profile.GetUsageSnapshot(p.Provider, p.Name)

				authStatus := "AUTH_OK"
				if !info.Authenticated {
					authStatus = "NO_AUTH"
				}

				capStr := "UNKNOWN"
				for _, w := range snap.Windows {
					if w.RemainingPercent != nil {
						capStr = fmt.Sprintf("%.0f%% left", *w.RemainingPercent)
						break
					}
				}
				if snap.Status == model.UsageRateLimited {
					capStr = "429 LIMITED"
				}

				freshStr := string(snap.Status)
				resetStr := "-"
				for _, w := range snap.Windows {
					if w.ResetDescription != "" {
						resetStr = w.ResetDescription
						break
					}
				}

				lines = append(lines, fmt.Sprintf("%s%-14s %-12s %-14s %-14s %s",
					activeMarker, p.Name, authStatus, capStr, freshStr, resetStr,
				))
			}
		}
		if found == 0 {
			lines = append(lines, "  (No profiles configured for this provider)")
		}
		return SlashResult{Intercepted: true, Response: strings.Join(lines, "\n") + "\n"}

	case "usage":
		snap := profile.GetUsageSnapshot(session.ProviderID, session.ProfileID)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\n=== Quota Snapshot: %s:%s ===\n", strings.ToUpper(session.ProviderID), session.ProfileID))
		sb.WriteString(fmt.Sprintf("Status:       %s\n", snap.Status))
		sb.WriteString(fmt.Sprintf("Source:       %s\n", snap.Source))
		if len(snap.Windows) > 0 {
			for _, w := range snap.Windows {
				remStr := "UNKNOWN"
				if w.RemainingPercent != nil {
					remStr = fmt.Sprintf("%.1f%%", *w.RemainingPercent)
				}
				sb.WriteString(fmt.Sprintf("Window [%s]:  Remaining: %s | Reset: %s\n", w.Kind, remStr, w.ResetDescription))
			}
		} else {
			sb.WriteString("Remaining:    UNKNOWN (No authentic quota metric exposed)\n")
		}
		if !snap.FetchedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("Last Check:   %s (%s ago)\n", snap.FetchedAt.Format(time.RFC3339), time.Since(snap.FetchedAt).Round(time.Second)))
		}
		return SlashResult{Intercepted: true, Response: sb.String() + "\n"}

	case "detach":
		return SlashResult{
			Intercepted: true,
			Action:      "detach",
			Response:    "\n[Nexus Control] Detached from runtime session. Host process remains running in background.\n",
		}

	case "stop":
		return SlashResult{
			Intercepted: true,
			Action:      "stop",
			Response:    "\n[Nexus Control] Stopping runtime session...\n",
		}

	case "handoff":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    fmt.Sprintf("Usage: %s handoff <target-profile> (e.g. %s handoff codex:work or %s handoff work)\n", prefix, prefix, prefix),
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "handoff",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[Nexus Control] Initiating same-provider account handoff to %s...\n", target),
		}

	case "continue":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    fmt.Sprintf("Usage: %s continue <target-provider[:profile]> (e.g. %s continue claude or %s continue claude:work)\n", prefix, prefix, prefix),
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "continue",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[Nexus Control] Creating cross-provider context envelope for %s...\n", target),
		}

	case "control", "ui":
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("\n[Nexus Control] To open the full Workspace OS, detach (%s detach) and run: nexus\n", prefix),
		}

	default:
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("Unknown command %q. Type %s help for available commands.\n", subcmd, prefix),
		}
	}
}

func fallbackUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "UNKNOWN"
	}
	return s
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func formatUsageSummary(snap model.UsageSnapshot) string {
	switch snap.Status {
	case model.UsageLive, model.UsageCached:
		for _, w := range snap.Windows {
			if w.RemainingPercent != nil {
				return fmt.Sprintf("%.0f%% remaining (%s)", *w.RemainingPercent, snap.Status)
			}
		}
		return string(snap.Status)
	case model.UsageRateLimited:
		return "RATE_LIMITED (429)"
	case model.UsageUnsupported:
		return "UNSUPPORTED"
	default:
		return "UNKNOWN"
	}
}
