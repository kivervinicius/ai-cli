package runner

import "time"

// State is an explicit durable state in the autonomous execution lifecycle.
type State string

const (
	StatePending              State = "PENDING"
	StateReady                State = "READY"
	StateAllocating           State = "ALLOCATING"
	StateCompiling            State = "COMPILING"
	StateExecuting            State = "EXECUTING"
	StateTesting              State = "TESTING"
	StateReviewing            State = "REVIEWING"
	StateVerifying            State = "VERIFYING"
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
	ID                 string   `json:"id"`
	PhaseID            string   `json:"phase_id"`
	Title              string   `json:"title"`
	Goal               string   `json:"goal"`
	Priority           string   `json:"priority"`
	Dependencies       []string `json:"dependencies,omitempty"`
	ParallelGroup      string   `json:"parallel_group,omitempty"`
	Role               string   `json:"role,omitempty"`
	TaskRequirements   string   `json:"task_requirements,omitempty"`
	AgentAllocation    string   `json:"agent_allocation,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
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

type PackageRun struct {
	ID                   string               `json:"id"`
	PackageID            string               `json:"package_id"`
	PhaseID              string               `json:"phase_id,omitempty"`
	Title                string               `json:"title"`
	Goal                 string               `json:"goal,omitempty"`
	Priority             string               `json:"priority,omitempty"`
	Role                 string               `json:"role,omitempty"`
	TaskRequirements     string               `json:"task_requirements,omitempty"`
	Dependencies         []string             `json:"dependencies,omitempty"`
	ParallelGroup        string               `json:"parallel_group,omitempty"`
	AcceptanceCriteria   []string             `json:"acceptance_criteria,omitempty"`
	State                State                `json:"state"`
	Attempt              int                  `json:"attempt"`
	AssignedAgent        string               `json:"assigned_agent"`
	AssignedRuntime      string               `json:"assigned_runtime,omitempty"`
	Workspace            string               `json:"workspace,omitempty"`
	PromptVersionID      string               `json:"prompt_version_id,omitempty"`
	CompiledPrompt       string               `json:"compiled_prompt,omitempty"`
	RemediationContext   string               `json:"remediation_context,omitempty"`
	RetryFrom            State                `json:"retry_from,omitempty"`
	LastFailureSignature string               `json:"last_failure_signature,omitempty"`
	NoProgressCount      int                  `json:"no_progress_count,omitempty"`
	Verdicts             []ReviewVerdict      `json:"verdicts,omitempty"`
	Verifications        []VerificationResult `json:"verifications,omitempty"`
	ErrorMessage         string               `json:"error_message,omitempty"`
	StartedAt            time.Time            `json:"started_at"`
	FinishedAt           *time.Time           `json:"finished_at,omitempty"`
}

type MissionRun struct {
	ID                  string               `json:"id"`
	PlanID              string               `json:"plan_id"`
	PlanRevision        int                  `json:"plan_revision"`
	ExecutionSnapshotID string               `json:"execution_snapshot_id,omitempty"`
	ProjectID           string               `json:"project_id"`
	Workspace           string               `json:"workspace"`
	State               State                `json:"state"`
	Contract            AutonomyContract     `json:"contract"`
	CurrentPkgIndex     int                  `json:"current_pkg_index"` // compatibility/UI hint only
	TotalIterations     int                  `json:"total_iterations"`
	PackageRuns         []PackageRun         `json:"package_runs"`
	LeaseOwner          string               `json:"lease_owner,omitempty"`
	LeaseToken          string               `json:"lease_token,omitempty"`
	LeaseExpiresAt      *time.Time           `json:"lease_expires_at,omitempty"`
	HeartbeatAt         *time.Time           `json:"heartbeat_at,omitempty"`
	PausedReason        string               `json:"paused_reason,omitempty"`
	ManualInterventions []ManualIntervention `json:"manual_interventions,omitempty"`
	StartedAt           time.Time            `json:"started_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	CompletedAt         *time.Time           `json:"completed_at,omitempty"`
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
