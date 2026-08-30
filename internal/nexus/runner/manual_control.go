package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
)

// ManualIntervention is durable evidence of a human takeover of one Mission
// package. It allows the autonomous worker to resume from the same Agent and
// worktree without pretending the manual edits were produced by the provider.
type ManualIntervention struct {
	ID                string     `json:"id"`
	PackageID         string     `json:"package_id"`
	AgentID           string     `json:"agent_id"`
	Workspace         string     `json:"workspace"`
	Reason            string     `json:"reason,omitempty"`
	BeforeFingerprint string     `json:"before_fingerprint,omitempty"`
	AfterFingerprint  string     `json:"after_fingerprint,omitempty"`
	ChangedPaths      []string   `json:"changed_paths,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// ManualControlPackage returns the package currently eligible for interactive
// takeover. Reviewers are never returned because they are not package owners.
func ManualControlPackage(run *MissionRun) *PackageRun {
	if run == nil || len(run.PackageRuns) == 0 {
		return nil
	}
	if run.CurrentPkgIndex >= 0 && run.CurrentPkgIndex < len(run.PackageRuns) {
		pkg := &run.PackageRuns[run.CurrentPkgIndex]
		if pkg.AssignedAgent != "" && pkg.State != StateVerified && pkg.State != StateFailed {
			return pkg
		}
	}
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.AssignedAgent == "" {
			continue
		}
		switch pkg.State {
		case StateVerified, StateFailed:
			continue
		default:
			return pkg
		}
	}
	return nil
}

// ManualControlAgentID returns the persistent implementer Agent that should be
// attached when a user takes over an autonomous run.
func ManualControlAgentID(run *MissionRun) string {
	pkg := ManualControlPackage(run)
	if pkg == nil {
		return ""
	}
	return pkg.AssignedAgent
}

// BeginManualIntervention records the exact package/worktree state before a
// human takes control. Only one unfinished intervention may exist per run.
func (r *MissionRunner) BeginManualIntervention(ctx context.Context, runID, agentID, packageID, workspace, beforeFingerprint, reason string) (*ManualIntervention, error) {
	run, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("cannot take control of terminal mission run %s", run.State)
	}
	for i := range run.ManualInterventions {
		if run.ManualInterventions[i].CompletedAt == nil {
			return nil, fmt.Errorf("manual intervention %s is already active", run.ManualInterventions[i].ID)
		}
	}
	matched := false
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.PackageID == packageID && pkg.AssignedAgent == agentID {
			matched = true
			if workspace == "" {
				workspace = pkg.Workspace
			}
			break
		}
	}
	if !matched {
		return nil, fmt.Errorf("agent %s is not assigned to package %s", agentID, packageID)
	}
	now := time.Now().UTC()
	entry := ManualIntervention{
		ID:                "manual_" + ids.NewRuntimeID(),
		PackageID:         packageID,
		AgentID:           agentID,
		Workspace:         workspace,
		Reason:            reason,
		BeforeFingerprint: beforeFingerprint,
		StartedAt:         now,
	}
	run.ManualInterventions = append(run.ManualInterventions, entry)
	run.UpdatedAt = now
	if err := r.repo.SaveRun(ctx, run); err != nil {
		return nil, err
	}
	copy := entry
	return &copy, nil
}

// CompleteManualIntervention persists after-state evidence. If the human
// changed implementation content while a package was in an implementation or
// validation stage, the autonomous flow resumes from TESTING rather than
// re-running the implementation prompt over manual edits.
func (r *MissionRunner) CompleteManualIntervention(ctx context.Context, runID, interventionID, afterFingerprint string, changedPaths []string) (*MissionRun, error) {
	run, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range run.ManualInterventions {
		if run.ManualInterventions[i].ID == interventionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("manual intervention %s not found", interventionID)
	}
	entry := &run.ManualInterventions[idx]
	if entry.CompletedAt != nil {
		return nil, fmt.Errorf("manual intervention %s is already complete", interventionID)
	}
	now := time.Now().UTC()
	entry.AfterFingerprint = afterFingerprint
	entry.ChangedPaths = append([]string(nil), changedPaths...)
	entry.CompletedAt = &now

	changed := len(changedPaths) > 0 || (entry.BeforeFingerprint != "" && afterFingerprint != "" && entry.BeforeFingerprint != afterFingerprint)
	if changed {
		for i := range run.PackageRuns {
			pkg := &run.PackageRuns[i]
			if pkg.PackageID != entry.PackageID || pkg.AssignedAgent != entry.AgentID {
				continue
			}
			switch pkg.State {
			case StateCompiling, StateExecuting, StateTesting, StateReviewing, StateVerifying, StateRemediating:
				pkg.State = StateTesting
				pkg.ErrorMessage = ""
				pkg.RemediationContext = ""
				pkg.RetryFrom = ""
			}
			break
		}
	}
	run.UpdatedAt = now
	if err := r.repo.SaveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}
