package exitcode

// Standard documented exit codes for the AI CLI Control Plane.
const (
	Success              = 0
	ProviderNotFound     = 10
	ProviderUnavailable  = 11
	ProfileNotFound      = 20
	ProfileUnavailable   = 21
	AuthRequired         = 30
	UsageUnknown         = 40
	RateLimited          = 41
	QuotaExhausted       = 42
	ConversationNotFound = 50
	ResumeUnsupported    = 51
	RuntimeError         = 60
	InvalidConfiguration = 70
)
