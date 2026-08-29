package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// MissionRunner orchestrates autonomous work plan execution (§Phase F & H).
type MissionRunner struct {
	leases   *LeaseManager
	retry    *RetryController
	verifier *VerificationEngine
}

func NewMissionRunner() *MissionRunner {
	return &MissionRunner{
		leases:   NewLeaseManager(),
		retry:    NewRetryController(),
		verifier: NewVerificationEngine(),
	}
}

// Leases returns the lease manager.
func (r *MissionRunner) Leases() *LeaseManager {
	return r.leases
}

// StartMissionRun initializes and registers a new autonomous execution run.
func (r *MissionRunner) StartMissionRun(
	ctx context.Context,
	plan store.WorkPlan,
	workspace string,
	contract AutonomyContract,
	defaultAgentID string,
) (*MissionRun, error) {
	runID := "run_" + ids.NewRuntimeID()
	now := time.Now().UTC()

	var pkgRuns []PackageRun
	for _, ph := range plan.Phases {
		for _, pkg := range ph.Packages {
			agent := pkg.AgentAllocation
			if agent == "" {
				agent = defaultAgentID
			}
			pkgRuns = append(pkgRuns, PackageRun{
				ID:            "pkgrun_" + ids.NewRuntimeID(),
				PackageID:     pkg.ID,
				Title:         pkg.Title,
				State:         StateReady,
				Attempt:       1,
				AssignedAgent: agent,
				StartedAt:     now,
			})
		}
	}

	if len(pkgRuns) == 0 {
		return nil, fmt.Errorf("plan %s contains no work packages to execute", plan.ID)
	}

	run := &MissionRun{
		ID:              runID,
		PlanID:          plan.ID,
		ProjectID:       plan.ProjectID,
		State:           StateExecuting,
		Contract:        contract,
		CurrentPkgIndex: 0,
		PackageRuns:     pkgRuns,
		StartedAt:       now,
		UpdatedAt:       now,
	}

	r.leases.RegisterRun(run)
	return run, nil
}

// ExecuteNextStep advances the current package in the mission run state machine.
func (r *MissionRunner) ExecuteNextStep(ctx context.Context, run *MissionRun, workspace string) (bool, error) {
	r.leases.Heartbeat(run.ID)
	run.UpdatedAt = time.Now().UTC()

	if run.CurrentPkgIndex >= len(run.PackageRuns) {
		run.State = StateCompleted
		now := time.Now().UTC()
		run.CompletedAt = &now
		r.leases.UnregisterRun(run.ID)
		return true, nil // All packages completed!
	}

	pkgRun := &run.PackageRuns[run.CurrentPkgIndex]

	switch pkgRun.State {
	case StateReady:
		pkgRun.State = StateExecuting

	case StateExecuting:
		// Transition to testing/verification
		pkgRun.State = StateTesting

	case StateTesting:
		// Run verification suite
		results := r.verifier.RunVerification(ctx, workspace, run.Contract.VerificationCommands)
		pkgRun.Verifications = append(pkgRun.Verifications, results...)

		allPassed := true
		for _, res := range results {
			if !res.Passed {
				allPassed = false
				break
			}
		}

		if allPassed {
			pkgRun.State = StateReviewing
		} else {
			pkgRun.State = StateRemediating
		}

	case StateRemediating:
		// Auto-remediation loop (§Phase H)
		canRetry, reason := r.retry.ShouldRetry(pkgRun, run.Contract)
		if !canRetry {
			pkgRun.State = StateFailed
			pkgRun.ErrorMessage = reason
			if run.Contract.EscalateOnFailure {
				run.State = StateEscalated
			} else {
				run.State = StateFailed
			}
			return false, fmt.Errorf("package failed: %s", reason)
		}

		pkgRun.Attempt++
		pkgRun.State = StateExecuting

	case StateReviewing:
		// Simulated or independent review verdict
		verdict := ReviewVerdict{
			Approved:        true,
			ReviewerAgentID: "reviewer-auto",
			ReviewedAt:      time.Now().UTC(),
		}
		pkgRun.Verdicts = append(pkgRun.Verdicts, verdict)

		if verdict.Approved {
			pkgRun.State = StateVerified
			now := time.Now().UTC()
			pkgRun.FinishedAt = &now
			run.CurrentPkgIndex++
		} else {
			pkgRun.State = StateRemediating
		}
	}

	return false, nil
}
