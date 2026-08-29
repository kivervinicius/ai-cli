package runner

// AutonomyContract defines the operational boundaries, auto-remediation policy,
// and escalation rules for autonomous execution (§Phase F & H).
type AutonomyContract struct {
	MaxRetries             int      `json:"max_retries"`              // Max attempts per WorkPackage (bounded, default 3)
	MaxTotalIterations     int      `json:"max_total_iterations"`     // Max loops across the mission
	AutoRemediate          bool     `json:"auto_remediate"`           // Automatically retry on review rejection
	RequireVerification    bool     `json:"require_verification"`     // Mandatory test/lint gate before package closure
	DisallowDestructiveGit bool     `json:"disallow_destructive_git"` // Refuse force push, branch delete, hard reset
	AllowedFilePatterns    []string `json:"allowed_file_patterns,omitempty"`
	VerificationCommands   []string `json:"verification_commands,omitempty"` // e.g. ["go test -race ./...", "npm test"]
	EscalateOnFailure      bool     `json:"escalate_on_failure"`             // Ask human when max retries exceeded
}

// DefaultAutonomyContract returns standard production-grade autonomy bounds.
func DefaultAutonomyContract() AutonomyContract {
	return AutonomyContract{
		MaxRetries:             3,
		MaxTotalIterations:     12,
		AutoRemediate:          true,
		RequireVerification:    true,
		DisallowDestructiveGit: true,
		VerificationCommands:   []string{"go test -race ./..."},
		EscalateOnFailure:      true,
	}
}
