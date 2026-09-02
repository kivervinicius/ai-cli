package runner

import "time"

// State is an explicit durable state in the autonomous execution lifecycle.
type State string

// DispatchState records the durable provider-dispatch boundary. INTENT is
// persisted before invoking the provider. If Nexus restarts with an INTENT and
// no completion evidence, the runner fails closed instead of dispatching the
// same Step a second time.
type DispatchState string

const (
	DispatchNone      DispatchState = ""
	DispatchIntent    DispatchState = "INTENT"
	DispatchCompleted DispatchState = "COMPLETED"
	DispatchFailed    DispatchState = "FAILED"
)

const (
	StatePending              State = "PENDING"
	StateReady                State = "READY"
	StateAllocating           State = "ALLOCATING"
	StateCompiling            State = "COMPILING"
	StateExecuting            State = "EXECUTING"
	StateTesting              State = "TESTING"
	StateReviewing            State = "REVIEWING"
	StateRemediating          State = "REMEDIATING"
	StateVerified             State = "VERIFIED"
	StateCompletedVerified    State = "COMPLETED_VERIFIED"
	StateFailed               State = "FAILED"
	StateFailedNoProgress     State = "FAILED_NO_PROGRESS"
	StateFailedBudgetExceeded State = "FAILED_BUDGET_EXCEEDED"
	StateFailedVerification   State = "FAILED_VERIFICATION"
	StateBlockedNeedsUser     State = "BLOCKED_NEEDS_USER"
	StateEscalated            State = "ESCALATED" // compatibility for older clients
	StatePaused               State = "PAUSED"
	StateCanceledByUser       State = "CANCELED_BY_USER"
	// StateCompleted is retained as an alias for old integrations.
	StateCompleted = StateCompletedVerified
)

// PlanSpec is the immutable execution input consumed by runner. It deliberately
// does not import the SQLite store package, keeping the state machine testable.
type PlanSpec struct {
	ID                  string        `json:"id"`
	ProjectID           string        `json:"project_id"`
	Revision            int           `json:"revision"`
	ExecutionSnapshotID string        `json:"execution_snapshot_id,omitempty"`
	Packages            []PackageSpec `json:"packages"`
}

type PackageSpec struct {
	ID                       string   `json:"id"`
	PhaseID                  string   `json:"phase_id"`
	Title                    string   `json:"title"`
	Goal                     string   `json:"goal"`
	Priority                 string   `json:"priority"`
	Dependencies             []string `json:"dependencies,omitempty"`
	ParallelGroup            string   `json:"parallel_group,omitempty"`
	Role                     string   `json:"role,omitempty"`
	TaskRequirements         string   `json:"task_requirements,omitempty"`
	AgentAllocation          string   `json:"agent_allocation,omitempty"`
	AssignmentStrategy       string   `json:"assignment_strategy,omitempty"`
	ResourcePolicy           string   `json:"resource_policy,omitempty"`
	Provider                 string   `json:"provider,omitempty"`
	Profile                  string   `json:"profile,omitempty"`
	MaestroSkills            []string `json:"maestro_skills,omitempty"`
	RelevantPaths            []string `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string `json:"acceptance_criteria,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
}

// ReviewVerdict records independent evaluation evidence. A verdict without a
// reviewer identity is never accepted by MissionRunner.
type ReviewVerdict struct {
	Approved        bool      `json:"approved"`
	ReviewerAgentID string    `json:"reviewer_agent_id"`
	Findings        []string  `json:"findings,omitempty"`
	RemediationTips []string  `json:"remediation_tips,omitempty"`
	ReviewedAt      time.Time `json:"reviewed_at"`
}

type VerificationResult struct {
	Command       string    `json:"command"`
	Passed        bool      `json:"passed"`
	ExitCode      int       `json:"exit_code"`
	OutputSnippet string    `json:"output_snippet"`
	DurationMs    int64     `json:"duration_ms"`
	VerifiedAt    time.Time `json:"verified_at"`
}

// ContextCapsule is the bounded typed context frozen for one Flow Step before
// prompt compilation/provider dispatch. It intentionally has no raw transcript
// or whole-repository content field.
type ContextCapsule struct {
	ID                        string             `json:"id"`
	RunID                     string             `json:"run_id"`
	ProjectID                 string             `json:"project_id"`
	FlowID                    string             `json:"flow_id"`
	FlowRevision              int                `json:"flow_revision"`
	Branch                    string             `json:"branch,omitempty"`
	Head                      string             `json:"head,omitempty"`
	DirtyFingerprint          string             `json:"dirty_fingerprint,omitempty"`
	Step                      ContextCapsuleStep `json:"step"`
	RelevantPaths             []string           `json:"relevant_paths,omitempty"`
	DurableContextRefs        []string           `json:"durable_context_refs,omitempty"`
	DependencyReceipts        []WorkReceipt      `json:"dependency_receipts,omitempty"`
	MaestroSkills             []string           `json:"maestro_skills,omitempty"`
	AcceptanceCriteria        []string           `json:"acceptance_criteria,omitempty"`
	Constraints               []string           `json:"constraints,omitempty"`
	BaselineWorkspaceSnapshot map[string]string  `json:"baseline_workspace_snapshot,omitempty"`
	CreatedAt                 time.Time          `json:"created_at"`
}

type ContextCapsuleStep struct {
	ID                       string   `json:"id"`
	Title                    string   `json:"title"`
	Goal                     string   `json:"goal,omitempty"`
	Role                     string   `json:"role,omitempty"`
	Dependencies             []string `json:"dependencies,omitempty"`
	AssignmentStrategy       string   `json:"assignment_strategy,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
}

// WorkReceipt is factual completion/failure evidence. Empty evidence remains
// empty; provider output is never promoted to proof merely because it says done.
type WorkReceipt struct {
	ID              string               `json:"id"`
	RunID           string               `json:"run_id"`
	StepID          string               `json:"step_id"`
	Status          string               `json:"status"`
	Summary         string               `json:"summary"`
	ChangedFiles    []string             `json:"changed_files"`
	Commands        []string             `json:"commands"`
	Tests           []VerificationResult `json:"tests"`
	Decisions       []string             `json:"decisions"`
	Artifacts       []string             `json:"artifacts"`
	RemainingIssues []string             `json:"remaining_issues"`
	Verification    []VerificationResult `json:"verification"`
	AgentID         string               `json:"agent_id,omitempty"`
	BaseRevision    string               `json:"base_revision,omitempty"`
	ResultRevision  string               `json:"result_revision,omitempty"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
}

type PackageRun struct {
	ID                       string               `json:"id"`
	PackageID                string               `json:"package_id"`
	PhaseID                  string               `json:"phase_id,omitempty"`
	Title                    string               `json:"title"`
	Goal                     string               `json:"goal,omitempty"`
	Priority                 string               `json:"priority,omitempty"`
	Role                     string               `json:"role,omitempty"`
	TaskRequirements         string               `json:"task_requirements,omitempty"`
	Dependencies             []string             `json:"dependencies,omitempty"`
	ParallelGroup            string               `json:"parallel_group,omitempty"`
	AssignmentStrategy       string               `json:"assignment_strategy,omitempty"`
	ResourcePolicy           string               `json:"resource_policy,omitempty"`
	Provider                 string               `json:"provider,omitempty"`
	Profile                  string               `json:"profile,omitempty"`
	MaestroSkills            []string             `json:"maestro_skills,omitempty"`
	RelevantPaths            []string             `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string             `json:"acceptance_criteria,omitempty"`
	VerificationRequirements []string             `json:"verification_requirements,omitempty"`
	ContextCapsule           *ContextCapsule      `json:"context_capsule,omitempty"`
	WorkReceipt              *WorkReceipt         `json:"work_receipt,omitempty"`
	State                    State                `json:"state"`
	Attempt                  int                  `json:"attempt"`
	AssignedAgent            string               `json:"assigned_agent"`
	AssignedRuntime          string               `json:"assigned_runtime,omitempty"`
	DispatchID               string               `json:"dispatch_id,omitempty"`
	DispatchState            DispatchState        `json:"dispatch_state,omitempty"`
	DispatchStartedAt        *time.Time           `json:"dispatch_started_at,omitempty"`
	DispatchFinishedAt       *time.Time           `json:"dispatch_finished_at,omitempty"`
	Workspace                string               `json:"workspace,omitempty"`
	PromptVersionID          string               `json:"prompt_version_id,omitempty"`
	CompiledPrompt           string               `json:"compiled_prompt,omitempty"`
	RemediationContext       string               `json:"remediation_context,omitempty"`
	RetryFrom                State                `json:"retry_from,omitempty"`
	LastFailureSignature     string               `json:"last_failure_signature,omitempty"`
	NoProgressCount          int                  `json:"no_progress_count,omitempty"`
	Verdicts                 []ReviewVerdict      `json:"verdicts,omitempty"`
	Verifications            []VerificationResult `json:"verifications,omitempty"`
	ErrorMessage             string               `json:"error_message,omitempty"`
	StartedAt                time.Time            `json:"started_at"`
	FinishedAt               *time.Time           `json:"finished_at,omitempty"`
}

type MissionRun struct {
	ID                  string           `json:"id"`
	PlanID              string           `json:"plan_id"`
	PlanRevision        int              `json:"plan_revision"`
	ExecutionSnapshotID string           `json:"execution_snapshot_id,omitempty"`
	ProjectID           string           `json:"project_id"`
	Workspace           string           `json:"workspace"`
	State               State            `json:"state"`
	Contract            AutonomyContract `json:"contract"`
	CurrentPkgIndex     int              `json:"current_pkg_index"` // compatibility/UI hint only
	TotalIterations     int              `json:"total_iterations"`
	PackageRuns         []PackageRun     `json:"package_runs"`
	LeaseOwner          string           `json:"lease_owner,omitempty"`
	LeaseToken          string           `json:"lease_token,omitempty"`
	LeaseExpiresAt      *time.Time       `json:"lease_expires_at,omitempty"`
	HeartbeatAt         *time.Time       `json:"heartbeat_at,omitempty"`
	PausedReason        string           `json:"paused_reason,omitempty"`
	StartedAt           time.Time        `json:"started_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	CompletedAt         *time.Time       `json:"completed_at,omitempty"`
}

type AllocationResult struct {
	AgentID   string `json:"agent_id"`
	Workspace string `json:"workspace"`
}

type PromptArtifact struct {
	VersionID string `json:"version_id"`
	Content   string `json:"content"`
}

type ExecutionResult struct {
	RuntimeID string `json:"runtime_id"`
	Output    string `json:"output,omitempty"`
}
