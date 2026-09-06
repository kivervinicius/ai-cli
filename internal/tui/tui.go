package tui

import (
	"os"

	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

// ShowMenu launches the unified interactive Quotas & Runtimes TUI.
// This bridges legacy invocations into the modern unified usage interface.
func ShowMenu() (*SelectionResult, error) {
	cfg, _ := config.LoadConfig()
	profs, _ := profile.List()

	accs := make(map[string]model.AccountInfo)
	rows := make([]UsageTableRow, 0, len(profs))

	for _, p := range profs {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		accs[p.Provider+":"+p.Name] = acc
		qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)

		modelName := ""
		if len(qv.ModelGroups) > 0 {
			modelName = qv.ModelGroups[0].Name
		}
		rows = append(rows, UsageTableRow{
			Provider:  p.Provider,
			Profile:   p.Name,
			Account:   acc.Email,
			Plan:      acc.Plan,
			ModelName: modelName,
			Status:    qv.Status,
			IsDefault: cfg.Defaults[p.Provider] == p.Name,
		})
	}

	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(30, cwd)

	opts := UnifiedUsageOptions{
		Rows:        rows,
		Sessions:    convs,
		Accounts:    accs,
		Defaults:    cfg.Defaults,
		Workspace:   cwd,
		InitialMode: ModeSafe,
	}

	return RunUnifiedUsage(opts)
}
