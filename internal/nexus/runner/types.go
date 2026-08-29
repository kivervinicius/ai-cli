package runner

import (
	"time"
)

// State represents the explicit phase in the autonomous execution state machine.
type State string

const (
	StateReady       State = "READY"
	StateAllocating  State = "ALLOCATING"
	StateCompiling   State = "COMPILING"
	StateExecuting   State = "EXECUTING"
	StateTesting     State = "TESTING"
	StateReviewing   State = "REVIEWING"
	StateRemediating State = "REMEDIATING"
	StateVerified    State = "VERIFIED"
	StateCompleted   State = "COMPLETED"
	StateFailed      State = "FAILED"
	StateEscalated   State = "ESCALATED"
	StatePaused      State = "PAUSED"
)

// ReviewVerdict records the independent evaluation of a work package implementation.
type ReviewVerdict struct {
	Approved        bool      `json:"approved"`
	ReviewerAgentID string    `json:"reviewer_agent_id"`
	Findings        []string  `json:"findings,omitempty"`
	RemediationTips []string  `json:"remediation_tips,omitempty"`
	ReviewedAt      time.Time `json:"reviewed_at"`
}

// VerificationResult holds the output of independent automated tests and gates.
type VerificationResult struct {
	Command       string    `json:"command"`
	Passed        bool      `json:"passed"`
	ExitCode      int       `json:"exit_code"`
	OutputSnippet string    `json:"output_snippet"`
	DurationMs    int64     `json:"duration_ms"`
	VerifiedAt    time.Time `json:"verified_at"`
}

// PackageRun tracks the execution, retries, and verification of one WorkPackage.
type PackageRun struct {
	ID              string               `json:"id"`
	PackageID       string               `json:"package_id"`
	Title           string               `json:"title"`
	State           State                `json:"state"`
	Attempt         int                  `json:"attempt"`
	AssignedAgent   string               `json:"assigned_agent"`
	AssignedRuntime string               `json:"assigned_runtime,omitempty"`
	CompiledPrompt  string               `json:"compiled_prompt,omitempty"`
	Verdicts        []ReviewVerdict      `json:"verdicts,omitempty"`
	Verifications   []VerificationResult `json:"verifications,omitempty"`
	ErrorMessage    string               `json:"error_message,omitempty"`
	StartedAt       time.Time            `json:"started_at"`
	FinishedAt      *time.Time           `json:"finished_at,omitempty"`
}

// MissionRun tracks the full end-to-end execution of a WorkPlan under an AutonomyContract.
type MissionRun struct {
	ID              string           `json:"id"`
	PlanID          string           `json:"plan_id"`
	ProjectID       string           `json:"project_id"`
	State           State            `json:"state"`
	Contract        AutonomyContract `json:"contract"`
	CurrentPkgIndex int              `json:"current_pkg_index"`
	PackageRuns     []PackageRun     `json:"package_runs"`
	StartedAt       time.Time        `json:"started_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`
}
