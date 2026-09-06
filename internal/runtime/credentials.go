package runtime

// CredentialIsolation defines the interface for running commands with isolated platform credentials/keyrings.
type CredentialIsolation interface {
	// WrapCommand wraps a binary and arguments to isolate credential stores (e.g. secret service / keychain).
	WrapCommand(bin string, args []string) (string, []string)
	// Platform returns the operating system platform name this isolator supports.
	Platform() string
	Capability() CredentialCapability
}

type CredentialCapabilityStatus string

const (
	CredentialSupported   CredentialCapabilityStatus = "SUPPORTED"
	CredentialUnsupported CredentialCapabilityStatus = "UNSUPPORTED"
	CredentialDegraded    CredentialCapabilityStatus = "DEGRADED"
)

type CredentialCapability struct {
	Status    CredentialCapabilityStatus `json:"status"`
	Mechanism string                     `json:"mechanism"`
	Reason    string                     `json:"reason,omitempty"`
}

// DefaultCredentialIsolator returns the platform-appropriate CredentialIsolation implementation.
func DefaultCredentialIsolator() CredentialIsolation {
	return newPlatformCredentialIsolation()
}
