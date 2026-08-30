package runner

// AutonomyContract defines operational boundaries, verification requirements,
// and bounded remediation for autonomous local engineering work.
type AutonomyContract struct {
	MaxRetries             int      `json:"max_retries"`
	MaxTotalIterations     int      `json:"max_total_iterations"`
	MaxNoProgress          int      `json:"max_no_progress"`
	PackageTimeoutSeconds  int      `json:"package_timeout_seconds"`
	AutoRemediate          bool     `json:"auto_remediate"`
	RequireVerification    bool     `json:"require_verification"`
	DisallowDestructiveGit bool     `json:"disallow_destructive_git"`
	AllowedFilePatterns    []string `json:"allowed_file_patterns,omitempty"`
	VerificationCommands   []string `json:"verification_commands,omitempty"`
	EscalateOnFailure      bool     `json:"escalate_on_failure"`
	AllowToolAutoApproval  bool     `json:"allow_tool_auto_approval"`
	AllowGitPush           bool     `json:"allow_git_push"`
	AllowDeploy            bool     `json:"allow_deploy"`
}

// DefaultAutonomyContract is intentionally local-first: it allows coding tools
// inside isolated workspaces but never authorizes push/deploy. Verification
// commands are detected from the target repository at Mission creation.
func DefaultAutonomyContract() AutonomyContract {
	return AutonomyContract{
		MaxRetries:             3,
		MaxTotalIterations:     120,
		MaxNoProgress:          2,
		PackageTimeoutSeconds:  3600,
		AutoRemediate:          true,
		RequireVerification:    true,
		DisallowDestructiveGit: true,
		VerificationCommands:   nil,
		EscalateOnFailure:      true,
		AllowToolAutoApproval:  true,
		AllowGitPush:           false,
		AllowDeploy:            false,
	}
}
