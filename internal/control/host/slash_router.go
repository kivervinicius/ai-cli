package host

import (
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
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

// RouteSlashCommand inspects terminal input lines and intercepts /ai commands.
func RouteSlashCommand(input string, session registry.RuntimeSession) SlashResult {
	trimmed := strings.TrimRight(input, "\r\n")

	// 1. Check for escape prefix "//ai"
	if strings.HasPrefix(trimmed, "//ai") {
		// Strip one slash: forward "/ai..." to the child process
		escaped := "/ai" + trimmed[4:]
		if strings.HasSuffix(input, "\n") {
			escaped += "\n"
		}
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: escaped,
		}
	}

	// 2. Check for reserved "/ai" command
	if !strings.HasPrefix(trimmed, "/ai") {
		// Normal input: forward untouched
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: input,
		}
	}

	// It is an /ai command: intercept completely
	parts := strings.Fields(trimmed)
	subcmd := ""
	if len(parts) > 1 {
		subcmd = parts[1]
	}

	switch subcmd {
	case "help", "?", "":
		resp := `
╔══════════════════════════════════════════════════════════════════╗
║               AI CONTROL — UNIVERSAL SLASH COMMANDS             ║
╠══════════════════════════════════════════════════════════════════╣
║  /ai status               Show current runtime status & quota   ║
║  /ai accounts             List available accounts for provider  ║
║  /ai usage                Display honest usage metrics          ║
║  /ai handoff <profile>    Same-provider account handoff         ║
║  /ai continue <provider>  Cross-provider context handoff        ║
║  /ai control              Open Control Center TUI instructions  ║
║  /ai detach               Detach from session (keeps alive)     ║
║  /ai stop                 Gracefully terminate runtime session  ║
║  //ai <text>              Escape prefix to send literal /ai     ║
╚══════════════════════════════════════════════════════════════════╝
`
		return SlashResult{Intercepted: true, Response: resp}

	case "status":
		resp := fmt.Sprintf(`
┌─ AI CONTROL STATUS ──────────────────────────────────────────┐
│  Runtime ID:       %-41s │
│  Provider:         %-41s │
│  Profile:          %-41s │
│  Provider Session: %-41s │
│  State:            %-41s │
│  Control Level:    %-41s │
│  Workspace:        %-41s │
└──────────────────────────────────────────────────────────────┘
`, session.RuntimeID, session.ProviderID, session.ProfileID, session.ProviderSessionID, session.State, session.ControlLevel, session.Workspace)
		return SlashResult{Intercepted: true, Response: resp}

	case "accounts":
		profs, _ := profile.List()
		var lines []string
		lines = append(lines, fmt.Sprintf("=== Accounts for Provider: %s ===", strings.ToUpper(session.ProviderID)))
		for _, p := range profs {
			if p.Provider == session.ProviderID {
				activeMarker := "  "
				if p.Name == session.ProfileID {
					activeMarker = "> "
				}
				info := profile.GetAccountInfo(p.Provider, p.Name)
				acc := info.Email
				if acc == "" {
					acc = info.Status
				}
				lines = append(lines, fmt.Sprintf("%s%-18s %-25s %s", activeMarker, p.Name, acc, info.Plan))
			}
		}
		if len(lines) == 1 {
			lines = append(lines, "  (No other profiles configured)")
		}
		return SlashResult{Intercepted: true, Response: strings.Join(lines, "\n") + "\n"}

	case "usage":
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("AI Control: Quota tracking active for %s:%s\n", session.ProviderID, session.ProfileID),
		}

	case "detach":
		return SlashResult{
			Intercepted: true,
			Action:      "detach",
			Response:    "\n[AI Control] Detached from runtime session. Process remains running.\n",
		}

	case "stop":
		return SlashResult{
			Intercepted: true,
			Action:      "stop",
			Response:    "\n[AI Control] Stopping runtime session...\n",
		}

	case "handoff":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    "Usage: /ai handoff <target-profile> (e.g. /ai handoff codex:personal or /ai handoff personal)\n",
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "handoff",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[AI Control] Initiating same-provider account handoff to %s...\n", target),
		}

	case "continue":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    "Usage: /ai continue <target-provider[:profile]> (e.g. /ai continue claude or /ai continue claude:work)\n",
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "continue",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[AI Control] Creating cross-provider context envelope for %s...\n", target),
		}

	case "control", "ui":
		return SlashResult{
			Intercepted: true,
			Response:    "\n[AI Control] To open the full Control Center, detach (/ai detach) and run: ai control\n",
		}

	default:
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("Unknown command %q. Type /ai help for available commands.\n", subcmd),
		}
	}
}
