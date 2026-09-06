package flags

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMergedHelp formats a combined help view showing the Nexus Canonical Aliases
// applicable to providerID followed by the official native help text.
func RenderMergedHelp(providerID string, nativeHelp string) string {
	providerID = strings.ToLower(strings.TrimSpace(providerID))

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("================================================================================\n")
	fmt.Fprintf(&sb, "  ⚡ Nexus · Canonical Aliases for %s\n", strings.ToUpper(providerID))
	sb.WriteString("================================================================================\n")
	sb.WriteString("Nexus translates these cross-provider canonical flags into native CLI options.\n")
	sb.WriteString("Native CLI flags are always supported 100% and take precedence.\n\n")

	type aliasRow struct {
		alias       string
		description string
		nativeArgs  string
	}

	var rows []aliasRow
	for alias, def := range BuiltinAliases {
		if native, ok := def.ProviderFlags[providerID]; ok && len(native) > 0 {
			rows = append(rows, aliasRow{
				alias:       alias,
				description: def.Description,
				nativeArgs:  strings.Join(native, " "),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].alias < rows[j].alias
	})

	fmt.Fprintf(&sb, "  %-18s  %-32s  %s\n", "CANONICAL ALIAS", "NATIVE CLI TRANSLATION", "DESCRIPTION")
	sb.WriteString("  " + strings.Repeat("─", 76) + "\n")

	for _, r := range rows {
		fmt.Fprintf(&sb, "  %-18s  %-32s  %s\n", r.alias, r.nativeArgs, r.description)
	}

	sb.WriteString("\nCustom aliases can be defined in ~/.config/nexus/config.json under \"flag_aliases\".\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	fmt.Fprintf(&sb, "  Official %s CLI Help:\n", strings.ToUpper(providerID))
	sb.WriteString("--------------------------------------------------------------------------------\n\n")

	if strings.TrimSpace(nativeHelp) != "" {
		sb.WriteString(strings.TrimRight(nativeHelp, "\n"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("  (No native help output available from provider CLI binary)\n")
	}

	return sb.String()
}
