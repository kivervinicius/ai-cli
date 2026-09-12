package runner

import (
	"strconv"
	"time"
)

const defaultStallTimeout = 15 * time.Minute

var inFlightWatchStates = map[State]struct{}{
	StateAllocating:  {},
	StateExecuting:   {},
	StateCompiling:   {},
	StateTesting:     {},
	StateReviewing:   {},
	StateVerifying:   {},
	StateRemediating: {},
}

func StallTimeout(contract AutonomyContract) time.Duration {
	if contract.StallTimeoutSeconds < 0 {
		return 0
	}
	if contract.StallTimeoutSeconds == 0 {
		return defaultStallTimeout
	}
	return time.Duration(contract.StallTimeoutSeconds) * time.Second
}

func progressAnchor(run *MissionRun) time.Time {
	if run == nil {
		return time.Time{}
	}
	if !run.LastProgressAt.IsZero() {
		return run.LastProgressAt
	}
	return run.StartedAt
}

func runIsInFlight(run *MissionRun) bool {
	if run == nil {
		return false
	}
	if _, ok := inFlightWatchStates[run.State]; ok {
		return true
	}
	for i := range run.PackageRuns {
		if _, ok := inFlightWatchStates[run.PackageRuns[i].State]; ok {
			return true
		}
	}
	return false
}

// ApplyProgressWatchdog fails closed when an in-flight Mission has produced
// no durable progress within StallTimeout. Heartbeats and UpdatedAt are not
// progress. FAILED_NO_PROGRESS is not converted into NEEDS_YOU.
func ApplyProgressWatchdog(run *MissionRun, now time.Time) bool {
	if run == nil || isTerminalRunState(run.State) || run.State == StatePaused || run.State == StateBlockedNeedsUser {
		return false
	}
	timeout := StallTimeout(run.Contract)
	if timeout <= 0 {
		return false
	}
	if !runIsInFlight(run) {
		return false
	}
	anchor := progressAnchor(run)
	if anchor.IsZero() {
		anchor = now
		run.LastProgressAt = now
		return false
	}
	if now.Sub(anchor) < timeout {
		return false
	}
	run.State = StateFailedNoProgress
	run.PausedReason = "watchdog: no durable progress within stall timeout"
	run.UpdatedAt = now
	for i := range run.PackageRuns {
		if _, ok := inFlightWatchStates[run.PackageRuns[i].State]; ok {
			run.PackageRuns[i].State = StateFailed
			run.PackageRuns[i].ErrorMessage = run.PausedReason
			run.PackageRuns[i].FinishedAt = &now
		}
	}
	return true
}

func progressFingerprint(run *MissionRun) string {
	if run == nil {
		return ""
	}
	buf := string(run.State)
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		buf += "|" + pkg.PackageID + ":" + string(pkg.State) + ":" + string(pkg.DispatchState)
		buf += ":" + pkg.StrategyVariant
		if pkg.WorkReceipt != nil {
			buf += ":receipt"
		}
		buf += ":" + strconv.Itoa(pkg.NoProgressCount) + ":" + strconv.Itoa(len(pkg.Verifications))
	}
	if run.NeedsHuman != nil {
		buf += ":h" + run.NeedsHuman.ID
	}
	return buf
}
