package runner

// ManualControlAgentID returns the persistent implementer Agent that should be
// attached when a user takes over an autonomous run. It never returns a
// reviewer Agent because reviewers are not stored as package assignments.
func ManualControlAgentID(run *MissionRun) string {
	if run == nil || len(run.PackageRuns) == 0 {
		return ""
	}
	if run.CurrentPkgIndex >= 0 && run.CurrentPkgIndex < len(run.PackageRuns) {
		pkg := run.PackageRuns[run.CurrentPkgIndex]
		if pkg.AssignedAgent != "" && pkg.State != StateVerified && pkg.State != StateFailed {
			return pkg.AssignedAgent
		}
	}
	for _, pkg := range run.PackageRuns {
		if pkg.AssignedAgent == "" {
			continue
		}
		switch pkg.State {
		case StateVerified, StateFailed:
			continue
		default:
			return pkg.AssignedAgent
		}
	}
	return ""
}
