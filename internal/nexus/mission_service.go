package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/autonomyguard"
	"github.com/kivervinicius/ai-cli/internal/nexus/maestrogates"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// missionExecutionSnapshot is immutable input for one approved MissionRun.
// The live WorkPlan may continue evolving without changing an active run.
type missionExecutionSnapshot struct {
	Plan     store.WorkPlan          `json:"plan"`
	Contract runner.AutonomyContract `json:"contract"`
}

type missionWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func planToRunnerSpec(plan *store.WorkPlan, snapshotID string) (runner.PlanSpec, error) {
	if plan == nil || plan.ID == "" || plan.ProjectID == "" {
		return runner.PlanSpec{}, fmt.Errorf("valid work plan is required")
	}
	spec := runner.PlanSpec{ID: plan.ID, ProjectID: plan.ProjectID, Revision: plan.CurrentRevision, ExecutionSnapshotID: snapshotID}
	seen := map[string]bool{}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			if strings.TrimSpace(pkg.ID) == "" {
				return runner.PlanSpec{}, fmt.Errorf("work package id is required in phase %s", phase.ID)
			}
			if seen[pkg.ID] {
				return runner.PlanSpec{}, fmt.Errorf("duplicate work package id %s", pkg.ID)
			}
			seen[pkg.ID] = true
			spec.Packages = append(spec.Packages, runner.PackageSpec{
				ID: pkg.ID, PhaseID: phase.ID, Title: pkg.Title, Goal: pkg.Goal, Priority: pkg.Priority,
				Dependencies: append([]string(nil), pkg.Dependencies...), ParallelGroup: pkg.ParallelGroup,
				Role: pkg.Role, TaskRequirements: pkg.TaskRequirements, AgentAllocation: pkg.AgentAllocation,
				AssignmentStrategy: pkg.AssignmentStrategy, ResourcePolicy: pkg.ResourcePolicy, Provider: pkg.Provider, Profile: pkg.Profile,
				MaestroSkills: append([]string(nil), pkg.MaestroSkills...), RelevantPaths: append([]string(nil), pkg.RelevantPaths...),
				AcceptanceCriteria: append([]string(nil), pkg.AcceptanceCriteria...), VerificationRequirements: append([]string(nil), pkg.VerificationRequirements...),
			})
		}
	}
	if len(spec.Packages) == 0 {
		return runner.PlanSpec{}, fmt.Errorf("work plan %s has no packages", plan.ID)
	}
	return spec, nil
}

func freezePlanForExecution(ctx context.Context, n *Nexus, plan store.WorkPlan) (store.WorkPlan, error) {
	if err := validateFlowExecutionContract(plan); err != nil {
		return store.WorkPlan{}, err
	}
	for pi := range plan.Phases {
		for wi := range plan.Phases[pi].Packages {
			pkg := &plan.Phases[pi].Packages[wi]
			requested := uniqueStrings(append(append([]string(nil), pkg.MaestroGates...), pkg.MaestroSkills...))
			validated, err := n.validateMaestroGatesStrict(ctx, requested)
			if err != nil {
				return store.WorkPlan{}, fmt.Errorf("freeze Maestro skills for package %s: %w", pkg.ID, err)
			}
			// The immutable execution snapshot carries one validated skill set in
			// both fields so legacy prompt compilation and the Flow façade agree.
			pkg.MaestroGates = append([]string(nil), validated...)
			pkg.MaestroSkills = append([]string(nil), validated...)
		}
	}
	return plan, nil
}

func validateFlowExecutionContract(plan store.WorkPlan) error {
	if err := ValidateFlowDAG(FlowFromWorkPlan(plan)); err != nil {
		return fmt.Errorf("invalid Flow DAG: %w", err)
	}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			strategy := strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy))
			switch strategy {
			case "": // legacy WorkPlan compatibility
			case string(FlowAssignmentExisting):
				if strings.TrimSpace(pkg.AgentAllocation) == "" {
					return fmt.Errorf("Flow Step %s uses EXISTING assignment without AgentID", pkg.ID)
				}
			case string(FlowAssignmentCreate):
				if strings.TrimSpace(pkg.AgentAllocation) != "" {
					return fmt.Errorf("Flow Step %s uses CREATE assignment with an existing AgentID", pkg.ID)
				}
			case string(FlowAssignmentAuto):
				if strings.TrimSpace(pkg.AgentAllocation) != "" {
					return fmt.Errorf("Flow Step %s uses AUTO assignment with a fixed AgentID", pkg.ID)
				}
			default:
				return fmt.Errorf("Flow Step %s has unsupported assignment strategy %q", pkg.ID, pkg.AssignmentStrategy)
			}
			policy := strings.ToUpper(strings.TrimSpace(pkg.ResourcePolicy))
			switch policy {
			case "", string(PolicyBalanced), string(PolicyPreserveQuota), string(PolicyPreferProvider), string(PolicyManual):
			default:
				return fmt.Errorf("Flow Step %s has unsupported resource policy %q", pkg.ID, pkg.ResourcePolicy)
			}
			if policy == string(PolicyManual) && strings.TrimSpace(pkg.Provider) == "" && strings.TrimSpace(pkg.Profile) == "" {
				return fmt.Errorf("Flow Step %s MANUAL resource policy requires provider and/or profile restriction", pkg.ID)
			}
		}
	}
	return nil
}

func (n *Nexus) validateMaestroGatesStrict(ctx context.Context, gates []string) ([]string, error) {
	if len(gates) == 0 {
		return nil, nil
	}
	client := NewMaestroClient()
	status := client.Status()
	var catalog []string
	if status.Capabilities != nil {
		catalog = status.Capabilities.Skills
	}
	var cause error
	if status.Error != "" {
		cause = fmt.Errorf("%s", status.Error)
	}
	return maestrogates.ValidateStrict(gates, status.Available, catalog, cause)
}

// StartMissionRun binds an immutable plan revision/snapshot and starts the
// durable state machine. autonomous=true starts a background worker immediately.
func (n *Nexus) StartMissionRun(ctx context.Context, planID, defaultAgentID string, contract runner.AutonomyContract, autonomous bool) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(plan.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("resolve mission project: %w", err)
	}
	revision, err := st.GetPlanRevision(plan.ID, plan.CurrentRevision)
	if err != nil {
		return nil, fmt.Errorf("resolve immutable plan revision %d: %w", plan.CurrentRevision, err)
	}

	contract = normalizeAutonomyContract(contract, project.CanonicalPath)
	frozenPlan, err := freezePlanForExecution(ctx, n, *plan)
	if err != nil {
		return nil, err
	}
	envelope, err := json.Marshal(missionExecutionSnapshot{Plan: frozenPlan, Contract: contract})
	if err != nil {
		return nil, fmt.Errorf("encode mission execution snapshot: %w", err)
	}
	snapshot, err := st.CreateExecutionSnapshot(plan.ID, revision.ID, string(envelope))
	if err != nil {
		return nil, fmt.Errorf("create immutable execution snapshot: %w", err)
	}
	spec, err := planToRunnerSpec(&frozenPlan, snapshot.ID)
	if err != nil {
		return nil, err
	}
	run, err := n.Runner().StartMissionRun(ctx, spec, project.CanonicalPath, contract, defaultAgentID)
	if err != nil {
		return nil, err
	}
	if autonomous {
		n.StartMissionWorker(run.ID)
	}
	return run, nil
}

func (n *Nexus) StartMissionRunApproved(ctx context.Context, planID string, approvedRevision int, defaultAgentID string, contract runner.AutonomyContract, autonomous bool) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	if plan.CurrentRevision != approvedRevision {
		return nil, fmt.Errorf("approved plan revision %d is stale; current is %d", approvedRevision, plan.CurrentRevision)
	}
	return n.startMissionRunAtRevision(ctx, plan, approvedRevision, defaultAgentID, contract, autonomous)
}

func (n *Nexus) startMissionRunAtRevision(ctx context.Context, plan *store.WorkPlan, revision int, defaultAgentID string, contract runner.AutonomyContract, autonomous bool) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	rev, err := st.GetPlanRevision(plan.ID, revision)
	if err != nil {
		return nil, err
	}
	var frozen store.WorkPlan
	if err := json.Unmarshal([]byte(rev.SnapshotJSON), &frozen); err != nil {
		return nil, err
	}
	contract = normalizeAutonomyContract(contract, project.CanonicalPath)
	env, err := json.Marshal(missionExecutionSnapshot{Plan: frozen, Contract: contract})
	if err != nil {
		return nil, err
	}
	snapshot, err := st.CreateExecutionSnapshot(plan.ID, rev.ID, string(env))
	if err != nil {
		return nil, err
	}
	spec, err := planToRunnerSpec(&frozen, snapshot.ID)
	if err != nil {
		return nil, err
	}
	run, err := n.Runner().StartMissionRun(ctx, spec, project.CanonicalPath, contract, defaultAgentID)
	if err != nil {
		return nil, err
	}
	if autonomous {
		n.StartMissionWorker(run.ID)
	}
	return run, nil
}

func normalizeAutonomyContract(contract runner.AutonomyContract, workspace string) runner.AutonomyContract {
	defaults := runner.DefaultAutonomyContract()
	if contract.MaxRetries <= 0 {
		contract.MaxRetries = defaults.MaxRetries
	}
	if contract.MaxTotalIterations <= 0 {
		contract.MaxTotalIterations = defaults.MaxTotalIterations
	}
	if contract.PackageTimeoutSeconds <= 0 {
		contract.PackageTimeoutSeconds = defaults.PackageTimeoutSeconds
	}
	if len(contract.VerificationCommands) == 0 && contract.RequireVerification {
		contract.VerificationCommands = detectVerificationCommands(workspace)
	}
	return contract
}

// detectVerificationCommands picks only commands backed by files/scripts that
// actually exist in the project. No Go-only verification is assumed.
func detectVerificationCommands(workspace string) []string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	var commands []string
	if fileExists(filepath.Join(workspace, "go.mod")) {
		commands = append(commands, "go test ./...")
	}
	if fileExists(filepath.Join(workspace, "package.json")) {
		commands = append(commands, packageVerificationCommands(workspace)...)
	}
	if fileExists(filepath.Join(workspace, "pyproject.toml")) || fileExists(filepath.Join(workspace, "pytest.ini")) || fileExists(filepath.Join(workspace, "requirements.txt")) {
		commands = append(commands, "python -m pytest")
	}
	if fileExists(filepath.Join(workspace, "Cargo.toml")) {
		commands = append(commands, "cargo test")
	}
	return uniqueStrings(commands)
}

func packageVerificationCommands(workspace string) []string {
	data, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	manager := "npm"
	if fileExists(filepath.Join(workspace, "yarn.lock")) {
		manager = "yarn"
	} else if fileExists(filepath.Join(workspace, "pnpm-lock.yaml")) {
		manager = "pnpm"
	} else if fileExists(filepath.Join(workspace, "bun.lock")) || fileExists(filepath.Join(workspace, "bun.lockb")) {
		manager = "bun"
	}
	var out []string
	for _, name := range []string{"typecheck", "lint", "test", "build"} {
		if _, ok := pkg.Scripts[name]; !ok {
			continue
		}
		switch manager {
		case "yarn", "bun":
			out = append(out, manager+" "+name)
		default:
			out = append(out, manager+" run "+name)
		}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func (n *Nexus) loadMissionSnapshot(run *runner.MissionRun) (*missionExecutionSnapshot, error) {
	if run == nil || strings.TrimSpace(run.ExecutionSnapshotID) == "" {
		return nil, fmt.Errorf("mission run has no immutable execution snapshot")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	snapshot, err := st.GetExecutionSnapshot(run.ExecutionSnapshotID)
	if err != nil {
		return nil, fmt.Errorf("load execution snapshot %s: %w", run.ExecutionSnapshotID, err)
	}
	var envelope missionExecutionSnapshot
	if err := json.Unmarshal([]byte(snapshot.StateJSON), &envelope); err != nil {
		return nil, fmt.Errorf("decode execution snapshot %s: %w", snapshot.ID, err)
	}
	if envelope.Plan.ID != run.PlanID || envelope.Plan.CurrentRevision != run.PlanRevision {
		return nil, fmt.Errorf("execution snapshot does not match mission run plan revision")
	}
	return &envelope, nil
}

// StartMissionWorker starts at most one autonomous worker per run in this process.
func (n *Nexus) StartMissionWorker(runID string) {
	if strings.TrimSpace(runID) == "" {
		return
	}
	n.workersMu.Lock()
	if n.workers == nil {
		n.workers = map[string]*missionWorker{}
	}
	if _, exists := n.workers[runID]; exists {
		n.workersMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	worker := &missionWorker{cancel: cancel, done: make(chan struct{})}
	n.workers[runID] = worker
	n.workersMu.Unlock()

	go func() {
		defer func() {
			n.workersMu.Lock()
			if current := n.workers[runID]; current == worker {
				delete(n.workers, runID)
			}
			n.workersMu.Unlock()
			close(worker.done)
		}()
		_, _ = n.Runner().RunToTerminal(ctx, runID)
	}()
}

// stopMissionWorker requests cancellation and waits briefly for the running
// provider/verification step to acknowledge it. This prevents a late worker
// save from overwriting PAUSED/CANCELED state.
func (n *Nexus) stopMissionWorker(runID string) error {
	n.workersMu.Lock()
	worker := n.workers[runID]
	n.workersMu.Unlock()
	if worker == nil {
		return nil
	}
	worker.cancel()
	select {
	case <-worker.done:
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mission worker did not stop within 15s")
	}
}

// RecoverMissionRuns resumes durable non-terminal runs after Nexus restart.
func (n *Nexus) RecoverMissionRuns(ctx context.Context) error {
	runs, err := n.Runner().ListRuns(ctx)
	if err != nil {
		return err
	}
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].StartedAt.Before(runs[j].StartedAt) })
	for _, run := range runs {
		if run == nil || runner.IsTerminalState(run.State) || run.State == runner.StatePaused || run.State == runner.StateBlockedNeedsUser {
			continue
		}
		n.StartMissionWorker(run.ID)
	}
	return nil
}

func (n *Nexus) PauseMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := n.stopMissionWorker(runID); err != nil {
		return nil, err
	}
	return n.Runner().PauseRun(ctx, runID, reason)
}

func (n *Nexus) ResumeMissionRun(ctx context.Context, runID string) (*runner.MissionRun, error) {
	run, err := n.Runner().ResumeRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	n.StartMissionWorker(runID)
	return run, nil
}

// TakeControlMissionRun pauses autonomous execution and starts an interactive
// supervised runtime for the package's persistent implementer Agent on the same
// isolated worktree. Mission state remains PAUSED until ReturnMissionRun.
func (n *Nexus) TakeControlMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	run, err := n.PauseMissionRun(ctx, runID, reason)
	if err != nil {
		return nil, err
	}
	agentID := runner.ManualControlAgentID(run)
	if agentID == "" {
		return run, nil
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, fmt.Errorf("resolve takeover agent: %w", err)
	}
	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return nil, fmt.Errorf("resolve takeover agent config: %w", err)
	}
	if cfg.Provider == "" || cfg.Profile == "" {
		return nil, fmt.Errorf("takeover agent %s has no provider/profile allocation", agentID)
	}
	if _, err := n.StartAgent(ctx, agentID, cfg.Provider, cfg.Profile); err != nil {
		return nil, fmt.Errorf("start interactive takeover runtime: %w", err)
	}
	return n.Runner().GetRun(ctx, runID)
}

// ReturnMissionRun stops the interactive takeover runtime before resuming the
// autonomous worker. This prevents the headless executor from racing a live
// human-controlled provider process on the same Agent/worktree.
func (n *Nexus) ReturnMissionRun(ctx context.Context, runID string) (*runner.MissionRun, error) {
	run, err := n.Runner().GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	agentID := runner.ManualControlAgentID(run)
	if agentID != "" {
		if err := n.StopAgent(ctx, agentID); err != nil {
			return nil, fmt.Errorf("stop interactive takeover runtime: %w", err)
		}
	}
	if len(run.Contract.AllowedFilePatterns) > 0 {
		snapshot, snapErr := n.loadMissionSnapshot(run)
		if snapErr != nil {
			return nil, snapErr
		}
		_ = snapshot // Contract is already bound to run; loading proves immutable snapshot still exists.
		pkgWorkspace := run.Workspace
		for i := range run.PackageRuns {
			if run.PackageRuns[i].AssignedAgent == agentID && run.PackageRuns[i].Workspace != "" {
				pkgWorkspace = run.PackageRuns[i].Workspace
				break
			}
		}
		changed, guardErr := autonomyguard.GitChangedPaths(ctx, pkgWorkspace)
		if guardErr != nil {
			return nil, fmt.Errorf("validate manual takeover changes: %w", guardErr)
		}
		if guardErr := autonomyguard.ValidateAllowedChanges(changed, run.Contract.AllowedFilePatterns); guardErr != nil {
			return nil, fmt.Errorf("manual takeover violates autonomy contract: %w", guardErr)
		}
	}
	return n.ResumeMissionRun(ctx, runID)
}

func (n *Nexus) CancelMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := n.stopMissionWorker(runID); err != nil {
		return nil, err
	}
	return n.Runner().CancelRun(ctx, runID, reason)
}
