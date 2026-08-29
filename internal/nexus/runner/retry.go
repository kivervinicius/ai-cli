package runner

import (
	"fmt"
)

// RetryController manages bounded retry loops and auto-remediation decisions (§Phase F).
type RetryController struct{}

func NewRetryController() *RetryController {
	return &RetryController{}
}

// ShouldRetry checks whether a package run can be retried under the contract.
func (c *RetryController) ShouldRetry(run *PackageRun, contract AutonomyContract) (bool, string) {
	if run.Attempt >= contract.MaxRetries {
		return false, fmt.Sprintf("limite de tentativas esgotado (%d/%d)", run.Attempt, contract.MaxRetries)
	}
	if !contract.AutoRemediate {
		return false, "auto-remediação desativada pelo contrato de autonomia"
	}
	return true, ""
}
