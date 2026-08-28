package handoff

import "github.com/kivervinicius/ai-cli/internal/control/host"

func init() {
	host.PerformAccountHandoff = PerformAccountHandoff
	host.PerformContextHandoff = PerformContextHandoff
}
