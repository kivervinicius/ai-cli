package registry

import (
	"time"
)

type RuntimeState string

const (
	StateStarting RuntimeState = "STARTING"
	StateRunning  RuntimeState = "RUNNING"
	StateWaiting  RuntimeState = "WAITING"
	StateApproval RuntimeState = "APPROVAL"
	StateDetached RuntimeState = "DETACHED"
	StateHandoff  RuntimeState = "HANDOFF"
	StateStopping RuntimeState = "STOPPING"
	StateStopped  RuntimeState = "STOPPED"
	StateFailed   RuntimeState = "FAILED"
	StateStale    RuntimeState = "STALE"
)

type ControlLevel string

const (
	ControlLevelAPI      ControlLevel = "CONTROL_API"
	ControlLevelEvents   ControlLevel = "EVENTS"
	ControlLevelTerminal ControlLevel = "TERMINAL"
	ControlLevelProcess  ControlLevel = "PROCESS"
)

// RuntimeSession represents a managed execution session under AI Control.
// Note: Secrets, OAuth tokens, and API keys are NEVER stored in this struct or persisted.
type RuntimeSession struct {
	RuntimeID         string            `json:"runtime_id"`
	ProviderID        string            `json:"provider_id"`
	ProfileID         string            `json:"profile_id"`
	ProviderSessionID string            `json:"provider_session_id,omitempty"`
	Workspace         string            `json:"workspace"`
	PID               int               `json:"pid"`
	State             RuntimeState      `json:"state"`
	ControlLevel      ControlLevel      `json:"control_level"`
	StartedAt         time.Time         `json:"started_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	ControlEndpoint   string            `json:"control_endpoint"`
	ParentRuntimeID   string            `json:"parent_runtime_id,omitempty"`
	HandoffType       string            `json:"handoff_type,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// IsActive returns true if the runtime is in an operational lifecycle state.
func (s RuntimeSession) IsActive() bool {
	switch s.State {
	case StateStarting, StateRunning, StateWaiting, StateApproval, StateDetached, StateHandoff:
		return true
	default:
		return false
	}
}
