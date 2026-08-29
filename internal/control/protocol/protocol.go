package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

const ProtocolVersion = 1

type CommandType string

const (
	CmdPing             CommandType = "ping"
	CmdStatus           CommandType = "status"
	CmdMetadata         CommandType = "metadata"
	CmdAttach           CommandType = "attach"
	CmdDetach           CommandType = "detach"
	CmdResize           CommandType = "resize"
	CmdInput            CommandType = "input"
	CmdStop             CommandType = "stop"
	CmdTerminate        CommandType = "terminate"
	CmdHandoff          CommandType = "handoff"
	CmdContinue         CommandType = "continue"
	CmdEvents           CommandType = "events"
	CmdSlash            CommandType = "slash"
	CmdLeaseAcquire     CommandType = "lease_acquire"
	CmdLeaseRelease     CommandType = "lease_release"
	CmdRuntimeChanged   CommandType = "runtime_changed"   // Gate 4: notify WS clients of generation switch
	CmdAgentState       CommandType = "agent_state"       // Gate 4: push effective agent state
	CmdContinuityState  CommandType = "continuity_state"  // Gate 4: push continuity status
)

// Request is a versioned command request sent to a SessionHost.
type Request struct {
	Version   int             `json:"version"`
	ID        string          `json:"id,omitempty"`
	Command   CommandType     `json:"command"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Response is the structured reply from a SessionHost.
type Response struct {
	Version   int             `json:"version"`
	ID        string          `json:"id,omitempty"`
	OK        bool            `json:"ok"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// ResizePayload specifies terminal window dimensions.
type ResizePayload struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// InputPayload contains raw terminal input bytes.
type InputPayload struct {
	Data string `json:"data"` // Base64 or plain string
}

// SlashPayload contains an intercepted slash command.
type SlashPayload struct {
	RawCommand string `json:"raw_command"`
}

// HandoffPayload specifies target profile for same-provider handoff.
type HandoffPayload struct {
	TargetProfile string `json:"target_profile"`
}

// ContinuePayload specifies target provider & profile for cross-provider context handoff.
type ContinuePayload struct {
	TargetProvider string `json:"target_provider"`
	TargetProfile  string `json:"target_profile,omitempty"`
}

// RuntimeChangedPayload notifies WS clients that the agent's runtime generation changed (Gate 4).
type RuntimeChangedPayload struct {
	AgentID        string `json:"agent_id"`
	OldRuntimeID   string `json:"old_runtime_id,omitempty"`
	NewRuntimeID   string `json:"new_runtime_id"`
	Provider       string `json:"provider"`
	Profile        string `json:"profile"`
	Continuity     string `json:"continuity"`
}

// AgentStatePayload pushes the honest effective agent state to WS clients (Gate 4).
type AgentStatePayload struct {
	AgentID string `json:"agent_id"`
	State   string `json:"state"`
}

// ContinuityStatePayload pushes continuity status updates (Gate 4).
type ContinuityStatePayload struct {
	AgentID   string `json:"agent_id"`
	Continuity string `json:"continuity"`
}

// StatusData contains runtime status reported by SessionHost.
type StatusData struct {
	RuntimeID         string    `json:"runtime_id"`
	ProviderID        string    `json:"provider_id"`
	ProfileID         string    `json:"profile_id"`
	ProviderSessionID string    `json:"provider_session_id,omitempty"`
	Workspace         string    `json:"workspace"`
	PID               int       `json:"pid"`
	State             string    `json:"state"`
	ControlLevel      string    `json:"control_level"`
	StartedAt         time.Time `json:"started_at"`
	QuotaStatus       string    `json:"quota_status,omitempty"`
	QuotaPercent      float64   `json:"quota_percent,omitempty"`
}

// NewRequest creates a standard request.
func NewRequest(cmd CommandType, payload any) (Request, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Request{}, err
		}
		raw = b
	}
	return Request{
		Version:   ProtocolVersion,
		ID:        fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Command:   cmd,
		Payload:   raw,
		Timestamp: time.Now(),
	}, nil
}

// NewResponse creates a successful response.
func NewResponse(data any) (Response, error) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return Response{}, err
		}
		raw = b
	}
	return Response{
		Version:   ProtocolVersion,
		OK:        true,
		Data:      raw,
		Timestamp: time.Now(),
	}, nil
}

// NewErrorResponse creates an error response.
func NewErrorResponse(errMsg string) Response {
	return Response{
		Version:   ProtocolVersion,
		OK:        false,
		Error:     errMsg,
		Timestamp: time.Now(),
	}
}
