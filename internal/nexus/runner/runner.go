package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
)

var ErrDispatchOutcomeUnknown = errors.New("provider dispatch outcome is unknown")

// MissionRunner is a deterministic, durable state machine. Provider-specific
// work is delegated to PackageExecutor and every transition is persisted.
type MissionRunner struct {
	repo     RunRepository
	executor PackageExecutor
	retry    *RetryController
	verifier *VerificationEngine
	owner    string
	leaseTTL time.Duration
}

func NewMissionRunner(repo RunRepository, executor PackageExecutor) *MissionRunner {
	if repo == nil {
		repo = NewMemoryRunRepository()
	}
	return &MissionRunner{
		repo: repo, executor: executor, retry: NewRetryController(), verifier: NewVerificationEngine(),
		owner: "runner_" + ids.NewRuntimeID(), leaseTTL: 30 * time.Second,
	}
}

func (r *MissionRunner) StartMissionRun(ctx context.Context, plan PlanSpec, workspace string, contract AutonomyContract, defaultAgentID string) (*MissionRun, error) {
	if r.executor == nil {
		return nil, fmt.Errorf("mission runner requires a real package executor")
	}
	if plan.ID == "" || plan.ProjectID == "" {
		return nil, fmt.Errorf("plan id and project id are required")
	}
	if len(plan.Packages) == 0 {
		return nil, fmt.Errorf("plan %s contains no work packages to execute", plan.ID)
	}
	if contract.MaxRetries <= 0 {
		contract.MaxRetries = 1
	}
	if contract.MaxTotalIterations <= 0 {
		contract.MaxTotalIterations = 1
	}
	if contract.MaxNoProgress <= 0 {
		contract.MaxNoProgress = 2
	}

	now := time.Now().UTC()
	run := &MissionRun{ID: "run_" + ids.NewRuntimeID(), PlanID: plan.ID, PlanRevision: plan.Revision, ExecutionSnapshotID: plan.ExecutionSnapshotID, ProjectID: plan.ProjectID,
		Workspace: workspace, State: StateExecuting, Contract: contract, StartedAt: now, UpdatedAt: now}
	for _, spec := range plan.Packages {
		state := StatePending
		if len(spec.Dependencies) == 0 {
			state = StateReady
		}
		assignedAgent := spec.AgentAllocation
		// Legacy packages without an explicit Flow assignment retain the old
		// default-Agent fallback. CREATE/AUTO stay unassigned until allocation so
		// explicit approval cannot accidentally reuse one Agent/worktree.
		if assignedAgent == "" && spec.AssignmentStrategy == "" && spec.ParallelGroup == "" {
			assignedAgent = defaultAgentID
		}
		run.PackageRuns = append(run.PackageRuns, PackageRun{
			ID: "pkgrun_" + ids.NewRuntimeID(), PackageID: spec.ID, PhaseID: spec.PhaseID, Title: spec.Title, Goal: spec.Goal,
			Priority: spec.Priority, Role: spec.Role, TaskRequirements: spec.TaskRequirements, Dependencies: append([]string(nil), spec.Dependencies...), ParallelGroup: spec.ParallelGroup,
			AssignmentStrategy: spec.AssignmentStrategy, ResourcePolicy: spec.ResourcePolicy, Provider: spec.Provider, Profile: spec.Profile,
			MaestroSkills: append([]string(nil), spec.MaestroSkills...), RelevantPaths: append([]string(nil), spec.RelevantPaths...),
			AcceptanceCriteria: append([]string(nil), spec.AcceptanceCriteria...), VerificationRequirements: append([]string(nil), spec.VerificationRequirements...), State: state, Attempt: 1,
			AssignedAgent: assignedAgent, StartedAt: now,
		})
	}
	if err := validateDependencyGraph(run.PackageRuns); err != nil {
		return nil, err
	}
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func chooseAgent(packageAgent, fallback string) string {
	if packageAgent != "" {
		return packageAgent
	}
	return fallback
}

func (r *MissionRunner) GetRun(ctx context.Context, id string) (*MissionRun, error) {
	return r.repo.GetRun(ctx, id)
}
func (r *MissionRunner) ListRuns(ctx context.Context) ([]*MissionRun, error) {
	return r.repo.ListRuns(ctx)
}

func (r *MissionRunner) ExecuteNextStep(ctx context.Context, runID string) (*MissionRun, bool, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, false, err
	}
	r.hydrateEvidence(ctx, run)
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	// Keep fencing ownership alive while provider/test/review operations may run
	// for minutes. If renewal fails, cancel the in-flight operation and refuse
	// to persist stale state.
	leaseCtx, leaseCancel := context.WithCancel(ctx)
	defer leaseCancel()
	heartbeatErr := make(chan error, 1)
	heartbeatDone := make(chan struct{})
	heartbeatEvery := r.leaseTTL / 3
	if heartbeatEvery <= 0 {
		heartbeatEvery = 10 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(heartbeatEvery)
		defer ticker.Stop()
		defer close(heartbeatDone)
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticker.C:
				if renewErr := r.repo.RenewLease(context.Background(), runID, r.owner, leaseToken, r.leaseTTL); renewErr != nil {
					select {
					case heartbeatErr <- renewErr:
					default:
					}
					leaseCancel()
					return
				}
			}
		}
	}()
	defer func() { leaseCancel(); <-heartbeatDone }()

	if isTerminalRunState(run.State) {
		return run, run.State == StateCompletedVerified, nil
	}
	if run.State == StatePaused {
		return run, false, fmt.Errorf("mission run is paused: %s", run.PausedReason)
	}

	run.TotalIterations++
	if run.TotalIterations > run.Contract.MaxTotalIterations {
		run.State = StateFailedBudgetExceeded
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, fmt.Errorf("mission iteration budget exceeded (%d/%d)", run.TotalIterations, run.Contract.MaxTotalIterations)
	}
	refreshDependencyStates(run)
	if allPackagesVerified(run) {
		return r.completeRun(ctx, run)
	}

	pkg := nextPackage(run)
	if pkg == nil {
		run.State = StateBlockedNeedsUser
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, fmt.Errorf("no runnable package; dependency graph is blocked")
	}
	run.CurrentPkgIndex = packageIndex(run, pkg.PackageID)

	opCtx := leaseCtx
	cancelOperation := func() {}
	if run.Contract.PackageTimeoutSeconds > 0 {
		opCtx, cancelOperation = context.WithTimeout(leaseCtx, time.Duration(run.Contract.PackageTimeoutSeconds)*time.Second)
	}
	defer cancelOperation()

	switch pkg.State {
	case StateReady:
		pkg.State = StateAllocating
	case StateAllocating:
		allocation, allocErr := r.executor.Allocate(opCtx, run, pkg)
		if allocErr != nil {
			return r.packageFailureFrom(ctx, run, pkg, StateAllocating, fmt.Errorf("allocate package: %w", allocErr))
		}
		if allocation.AgentID == "" || allocation.Workspace == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateAllocating, fmt.Errorf("allocator returned incomplete assignment"))
		}
		pkg.AssignedAgent, pkg.Workspace = allocation.AgentID, allocation.Workspace
		pkg.State = StateCompiling
	case StateCompiling:
		if capsuleErr := r.ensureContextCapsule(opCtx, run, pkg); capsuleErr != nil {
			if errors.Is(capsuleErr, ErrInfrastructureFailure) {
				// Evidence persistence is an infrastructure boundary. Fail closed
				// without retrying implementation work or invoking a provider.
				pkg.State = StateFailed
				pkg.ErrorMessage = "SCHEMA_UNAVAILABLE"
				run.State = StateFailed
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, capsuleErr
			}
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("prepare context capsule: %w", capsuleErr))
		}
		artifact, compileErr := r.executor.Compile(opCtx, run, pkg)
		if compileErr != nil {
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("compile package prompt: %w", compileErr))
		}
		if artifact.VersionID == "" || artifact.Content == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("prompt compiler returned no immutable artifact"))
		}
		pkg.PromptVersionID, pkg.CompiledPrompt = artifact.VersionID, artifact.Content
		pkg.State = StateExecuting
	case StateExecuting:
		if pkg.ParallelGroup != "" {
			if err := r.executeParallelGroup(opCtx, run, pkg.ParallelGroup); err != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, err
			}
		} else {
			if err := r.executeOne(opCtx, run, pkg); err != nil {
				if errors.Is(err, ErrDispatchOutcomeUnknown) {
					run.State = StateBlockedNeedsUser
					run.UpdatedAt = time.Now().UTC()
					_ = r.saveRun(ctx, run)
					return run, false, err
				}
				if terminalErr := r.markRemediation(run, pkg, StateCompiling, err.Error()); terminalErr != nil {
					run.UpdatedAt = time.Now().UTC()
					_ = r.saveRun(ctx, run)
					return run, false, terminalErr
				}
			}
		}
	case StateTesting:
		verificationCommands := run.Contract.VerificationCommands
		if len(pkg.VerificationRequirements) > 0 {
			verificationCommands = pkg.VerificationRequirements
		}
		if run.Contract.RequireVerification && len(verificationCommands) == 0 {
			if terminalErr := r.markRemediation(run, pkg, StateCompiling, "Verification is required but no verification commands are configured."); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
			break
		}
		results := r.verifier.RunVerification(opCtx, pkg.Workspace, verificationCommands)
		pkg.Verifications = append(pkg.Verifications, results...)
		if verificationPassed(results, run.Contract.RequireVerification) {
			pkg.State = StateReviewing
		} else {
			if terminalErr := r.markRemediation(run, pkg, StateCompiling, verificationFailureContext(results)); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
		}
	case StateReviewing:
		verdict, reviewErr := r.executor.Review(opCtx, run, pkg)
		if reviewErr != nil {
			if terminalErr := r.markRemediation(run, pkg, StateReviewing, "Independent review failed: "+reviewErr.Error()); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
			break
		}
		if verdict.ReviewerAgentID == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateReviewing, fmt.Errorf("review verdict has no reviewer identity"))
		}
		if verdict.ReviewedAt.IsZero() {
			verdict.ReviewedAt = time.Now().UTC()
		}
		pkg.Verdicts = append(pkg.Verdicts, verdict)
		if verdict.Approved {
			pkg.State = StateVerified
			now := time.Now().UTC()
			pkg.FinishedAt = &now
			pkg.RemediationContext = ""
		} else {
			if terminalErr := r.markRemediation(run, pkg, StateCompiling, reviewFailureContext(verdict)); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
		}
	case StateRemediating:
		canRetry, reason := r.retry.ShouldRetry(pkg, run.Contract)
		if !canRetry {
			pkg.State = StateFailed
			pkg.ErrorMessage = reason
			if run.Contract.EscalateOnFailure {
				run.State = StateBlockedNeedsUser
			} else {
				run.State = StateFailedNoProgress
			}
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("package %s failed: %s", pkg.PackageID, reason)
		}
		pkg.Attempt++
		resetDispatch(pkg)
		next := pkg.RetryFrom
		if next == "" || next == StateRemediating {
			next = StateExecuting
		}
		pkg.State = next
	case StatePending:
		return r.packageFailureFrom(ctx, run, pkg, StatePending, fmt.Errorf("package selected before dependencies were verified"))
	}

	refreshDependencyStates(run)
	if allPackagesVerified(run) {
		return r.completeRun(ctx, run)
	}
	select {
	case hbErr := <-heartbeatErr:
		return run, false, fmt.Errorf("mission lease heartbeat lost: %w", hbErr)
	default:
	}
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	return run, false, nil
}

func (r *MissionRunner) packageFailureFrom(ctx context.Context, run *MissionRun, pkg *PackageRun, retryFrom State, err error) (*MissionRun, bool, error) {
	if terminalErr := r.markRemediation(run, pkg, retryFrom, err.Error()); terminalErr != nil {
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, terminalErr
	}
	run.UpdatedAt = time.Now().UTC()
	_ = r.saveRun(ctx, run)
	return run, false, err
}

func (r *MissionRunner) markRemediation(run *MissionRun, pkg *PackageRun, retryFrom State, failure string) error {
	failure = strings.TrimSpace(failure)
	if failure == "" {
		failure = "unknown package failure"
	}
	if pkg.LastFailureSignature == failure {
		pkg.NoProgressCount++
	} else {
		pkg.LastFailureSignature = failure
		pkg.NoProgressCount = 1
	}
	pkg.ErrorMessage = failure
	pkg.RemediationContext = failure
	pkg.RetryFrom = retryFrom
	pkg.State = StateRemediating
	limit := run.Contract.MaxNoProgress
	if limit <= 0 {
		limit = 2
	}
	if pkg.NoProgressCount >= limit {
		pkg.State = StateFailed
		run.State = StateFailedNoProgress
		return fmt.Errorf("package %s made no progress after %d identical failures: %s", pkg.PackageID, pkg.NoProgressCount, failure)
	}
	return nil
}

func (r *MissionRunner) completeRun(ctx context.Context, run *MissionRun) (*MissionRun, bool, error) {
	now := time.Now().UTC()
	run.State = StateCompletedVerified
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	return run, true, nil
}

func (r *MissionRunner) PauseRun(ctx context.Context, runID, reason string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("cannot pause terminal mission run %s", run.State)
	}
	run.State = StatePaused
	run.PausedReason = reason
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (r *MissionRunner) ResumeRun(ctx context.Context, runID string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if run.State != StatePaused && run.State != StateBlockedNeedsUser {
		return nil, fmt.Errorf("mission run %s cannot resume from %s", runID, run.State)
	}
	run.State = StateExecuting
	run.PausedReason = ""
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (r *MissionRunner) CancelRun(ctx context.Context, runID, reason string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if isTerminalRunState(run.State) {
		return run, nil
	}
	now := time.Now().UTC()
	run.State = StateCanceledByUser
	run.PausedReason = reason
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// RunToTerminal executes a durable mission autonomously until a terminal or
// human-blocked state is reached. Transient package errors are allowed to flow
// through the bounded remediation state machine; terminal failures are returned.
func (r *MissionRunner) RunToTerminal(ctx context.Context, runID string) (*MissionRun, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		run, done, stepErr := r.ExecuteNextStep(ctx, runID)
		if run == nil {
			return nil, stepErr
		}
		if done || run.State == StateCompletedVerified {
			return run, nil
		}
		if run.State == StatePaused || run.State == StateBlockedNeedsUser || isTerminalRunState(run.State) {
			if stepErr != nil {
				return run, stepErr
			}
			return run, fmt.Errorf("mission stopped in state %s", run.State)
		}
		// A package-level error is persisted as REMEDIATING; keep executing
		// while the contract still permits retries instead of aborting the run.
		if stepErr != nil {
			continue
		}
	}
}

func (r *MissionRunner) beginDispatch(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if pkg.DispatchState == DispatchIntent && pkg.DispatchID != "" {
		return fmt.Errorf("%w: package %s has unresolved dispatch %s; refusing duplicate provider execution", ErrDispatchOutcomeUnknown, pkg.PackageID, pkg.DispatchID)
	}
	now := time.Now().UTC()
	pkg.DispatchID = "dispatch_" + ids.NewRuntimeID()
	pkg.DispatchState = DispatchIntent
	pkg.DispatchStartedAt = &now
	pkg.DispatchFinishedAt = nil
	return r.saveRun(ctx, run)
}

func finishDispatch(pkg *PackageRun, state DispatchState) {
	now := time.Now().UTC()
	pkg.DispatchState = state
	pkg.DispatchFinishedAt = &now
}

func resetDispatch(pkg *PackageRun) {
	pkg.DispatchID = ""
	pkg.DispatchState = DispatchNone
	pkg.DispatchStartedAt = nil
	pkg.DispatchFinishedAt = nil
	pkg.AssignedRuntime = ""
}

func (r *MissionRunner) executeOne(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if err := r.beginDispatch(ctx, run, pkg); err != nil {
		run.State = StateBlockedNeedsUser
		pkg.ErrorMessage = err.Error()
		return err
	}
	outcome, err := r.executor.Execute(ctx, run, pkg, pkg.CompiledPrompt)
	if err != nil {
		finishDispatch(pkg, DispatchFailed)
		return fmt.Errorf("Execution failed: %w", err)
	}
	if outcome.RuntimeID == "" {
		finishDispatch(pkg, DispatchFailed)
		return fmt.Errorf("executor returned no runtime evidence")
	}
	pkg.AssignedRuntime = outcome.RuntimeID
	finishDispatch(pkg, DispatchCompleted)
	pkg.ErrorMessage = ""
	pkg.State = StateTesting
	return nil
}

// executeParallelGroup runs provider turns concurrently only after every runnable
// member of the group has reached EXECUTING. Earlier lifecycle stages remain
// deterministic and persisted one transition at a time.
func (r *MissionRunner) executeParallelGroup(ctx context.Context, run *MissionRun, group string) error {
	indexes := make([]int, 0)
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.ParallelGroup != group || pkg.State == StateVerified || pkg.State == StateFailed || pkg.State == StatePending {
			continue
		}
		if pkg.State != StateExecuting {
			return nil
		}
		indexes = append(indexes, i)
	}
	if len(indexes) == 0 {
		return nil
	}
	// Persist every dispatch intent before launching any provider goroutine.
	for _, idx := range indexes {
		pkg := &run.PackageRuns[idx]
		if pkg.DispatchState == DispatchIntent && pkg.DispatchID != "" {
			run.State = StateBlockedNeedsUser
			pkg.ErrorMessage = fmt.Sprintf("package %s has unresolved dispatch %s; refusing duplicate provider execution", pkg.PackageID, pkg.DispatchID)
			return fmt.Errorf("%w: %s", ErrDispatchOutcomeUnknown, pkg.ErrorMessage)
		}
		now := time.Now().UTC()
		pkg.DispatchID = "dispatch_" + ids.NewRuntimeID()
		pkg.DispatchState = DispatchIntent
		pkg.DispatchStartedAt = &now
		pkg.DispatchFinishedAt = nil
	}
	if err := r.saveRun(ctx, run); err != nil {
		return err
	}
	type result struct {
		index   int
		outcome ExecutionResult
		err     error
	}
	results := make(chan result, len(indexes))
	for _, idx := range indexes {
		idx := idx
		runCopy := *run
		runCopy.PackageRuns = append([]PackageRun(nil), run.PackageRuns...)
		go func() {
			pkg := &runCopy.PackageRuns[idx]
			outcome, err := r.executor.Execute(ctx, &runCopy, pkg, pkg.CompiledPrompt)
			results <- result{index: idx, outcome: outcome, err: err}
		}()
	}
	var firstErr error
	for range indexes {
		res := <-results
		pkg := &run.PackageRuns[res.index]
		if res.err != nil {
			finishDispatch(pkg, DispatchFailed)
			if err := r.markRemediation(run, pkg, StateCompiling, "Execution failed: "+res.err.Error()); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if res.outcome.RuntimeID == "" {
			finishDispatch(pkg, DispatchFailed)
			if err := r.markRemediation(run, pkg, StateCompiling, "executor returned no runtime evidence"); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		pkg.AssignedRuntime = res.outcome.RuntimeID
		finishDispatch(pkg, DispatchCompleted)
		pkg.ErrorMessage = ""
		pkg.State = StateTesting
	}
	return firstErr
}

func validateDependencyGraph(pkgs []PackageRun) error {
	known := map[string]bool{}
	for _, p := range pkgs {
		if p.PackageID == "" {
			return fmt.Errorf("package id is required")
		}
		if known[p.PackageID] {
			return fmt.Errorf("duplicate package id %s", p.PackageID)
		}
		known[p.PackageID] = true
	}
	for _, p := range pkgs {
		for _, dep := range p.Dependencies {
			if dep == p.PackageID {
				return fmt.Errorf("package %s depends on itself", p.PackageID)
			}
			if !known[dep] {
				return fmt.Errorf("package %s depends on unknown package %s", p.PackageID, dep)
			}
		}
	}
	// cycle check via DFS
	deps := map[string][]string{}
	for _, p := range pkgs {
		deps[p.PackageID] = p.Dependencies
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("dependency cycle includes %s", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range deps {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func refreshDependencyStates(run *MissionRun) {
	states := map[string]State{}
	for _, p := range run.PackageRuns {
		states[p.PackageID] = p.State
	}
	for i := range run.PackageRuns {
		p := &run.PackageRuns[i]
		if p.State != StatePending {
			continue
		}
		ready := true
		for _, dep := range p.Dependencies {
			if states[dep] != StateVerified {
				ready = false
				break
			}
		}
		if ready {
			p.State = StateReady
		}
	}
}
func nextPackage(run *MissionRun) *PackageRun {
	var selected *PackageRun
	selectedRank := int(^uint(0) >> 1)
	for i := range run.PackageRuns {
		p := &run.PackageRuns[i]
		if p.State == StateVerified || p.State == StatePending || p.State == StateFailed {
			continue
		}
		rank := packageStageRank(p.State)
		if selected == nil || rank < selectedRank {
			selected, selectedRank = p, rank
		}
	}
	return selected
}

func packageStageRank(state State) int {
	switch state {
	case StateReady:
		return 0
	case StateAllocating:
		return 1
	case StateCompiling:
		return 2
	case StateExecuting:
		return 3
	case StateTesting:
		return 4
	case StateReviewing:
		return 5
	case StateRemediating:
		return 6
	default:
		return 100
	}
}
func allPackagesVerified(run *MissionRun) bool {
	if len(run.PackageRuns) == 0 {
		return false
	}
	for _, p := range run.PackageRuns {
		if p.State != StateVerified {
			return false
		}
	}
	return true
}
func packageIndex(run *MissionRun, id string) int {
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == id {
			return i
		}
	}
	return -1
}
func verificationPassed(results []VerificationResult, required bool) bool {
	if required && len(results) == 0 {
		return false
	}
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
func verificationFailureContext(results []VerificationResult) string {
	for i := len(results) - 1; i >= 0; i-- {
		if !results[i].Passed {
			return fmt.Sprintf("Verification failed: %s (exit %d)\n%s", results[i].Command, results[i].ExitCode, results[i].OutputSnippet)
		}
	}
	return "Verification produced no passing evidence."
}
func reviewFailureContext(v ReviewVerdict) string {
	return fmt.Sprintf("Review rejected. Findings: %v. Remediation: %v", v.Findings, v.RemediationTips)
}

// IsTerminalState reports whether a mission cannot make further progress.
func IsTerminalState(s State) bool {
	return s == StateCompletedVerified || s == StateFailed || s == StateFailedNoProgress || s == StateFailedBudgetExceeded || s == StateFailedVerification || s == StateCanceledByUser
}

func isTerminalRunState(s State) bool { return IsTerminalState(s) }
