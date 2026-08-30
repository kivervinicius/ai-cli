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
	AllowExternalNetwork   bool     `json:"allow_external_network"`
	AllowSecretAccess      bool     `json:"allow_secret_access"`
	AllowPaidServices      bool     `json:"allow_paid_services"`
}

// AutonomyContractPatch is the HTTP/API representation used when callers want
// to override only selected autonomy boundaries. Pointer fields distinguish an
// omitted value from an explicit false/zero. Omitted fields inherit the safe
// defaults from DefaultAutonomyContract.
type AutonomyContractPatch struct {
	MaxRetries             *int      `json:"max_retries,omitempty"`
	MaxTotalIterations     *int      `json:"max_total_iterations,omitempty"`
	MaxNoProgress          *int      `json:"max_no_progress,omitempty"`
	PackageTimeoutSeconds  *int      `json:"package_timeout_seconds,omitempty"`
	AutoRemediate          *bool     `json:"auto_remediate,omitempty"`
	RequireVerification    *bool     `json:"require_verification,omitempty"`
	DisallowDestructiveGit *bool     `json:"disallow_destructive_git,omitempty"`
	AllowedFilePatterns    *[]string `json:"allowed_file_patterns,omitempty"`
	VerificationCommands   *[]string `json:"verification_commands,omitempty"`
	EscalateOnFailure      *bool     `json:"escalate_on_failure,omitempty"`
	AllowToolAutoApproval  *bool     `json:"allow_tool_auto_approval,omitempty"`
	AllowGitPush           *bool     `json:"allow_git_push,omitempty"`
	AllowDeploy            *bool     `json:"allow_deploy,omitempty"`
	AllowExternalNetwork   *bool     `json:"allow_external_network,omitempty"`
	AllowSecretAccess      *bool     `json:"allow_secret_access,omitempty"`
	AllowPaidServices      *bool     `json:"allow_paid_services,omitempty"`
}

// ApplyAutonomyContractPatch merges a sparse API patch onto the intentionally
// conservative local-first defaults. Dangerous permissions remain denied
// unless the caller explicitly enables them.
func ApplyAutonomyContractPatch(patch *AutonomyContractPatch) AutonomyContract {
	contract := DefaultAutonomyContract()
	if patch == nil {
		return contract
	}
	if patch.MaxRetries != nil && *patch.MaxRetries > 0 {
		contract.MaxRetries = *patch.MaxRetries
	}
	if patch.MaxTotalIterations != nil && *patch.MaxTotalIterations > 0 {
		contract.MaxTotalIterations = *patch.MaxTotalIterations
	}
	if patch.MaxNoProgress != nil && *patch.MaxNoProgress > 0 {
		contract.MaxNoProgress = *patch.MaxNoProgress
	}
	if patch.PackageTimeoutSeconds != nil && *patch.PackageTimeoutSeconds > 0 {
		contract.PackageTimeoutSeconds = *patch.PackageTimeoutSeconds
	}
	if patch.AutoRemediate != nil {
		contract.AutoRemediate = *patch.AutoRemediate
	}
	if patch.RequireVerification != nil {
		contract.RequireVerification = *patch.RequireVerification
	}
	if patch.DisallowDestructiveGit != nil {
		contract.DisallowDestructiveGit = *patch.DisallowDestructiveGit
	}
	if patch.AllowedFilePatterns != nil {
		contract.AllowedFilePatterns = append([]string(nil), (*patch.AllowedFilePatterns)...)
	}
	if patch.VerificationCommands != nil {
		contract.VerificationCommands = append([]string(nil), (*patch.VerificationCommands)...)
	}
	if patch.EscalateOnFailure != nil {
		contract.EscalateOnFailure = *patch.EscalateOnFailure
	}
	if patch.AllowToolAutoApproval != nil {
		contract.AllowToolAutoApproval = *patch.AllowToolAutoApproval
	}
	if patch.AllowGitPush != nil {
		contract.AllowGitPush = *patch.AllowGitPush
	}
	if patch.AllowDeploy != nil {
		contract.AllowDeploy = *patch.AllowDeploy
	}
	if patch.AllowExternalNetwork != nil {
		contract.AllowExternalNetwork = *patch.AllowExternalNetwork
	}
	if patch.AllowSecretAccess != nil {
		contract.AllowSecretAccess = *patch.AllowSecretAccess
	}
	if patch.AllowPaidServices != nil {
		contract.AllowPaidServices = *patch.AllowPaidServices
	}
	return contract
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
		AllowExternalNetwork:   false,
		AllowSecretAccess:      false,
		AllowPaidServices:      false,
	}
}
