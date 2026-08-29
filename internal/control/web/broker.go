package web

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
)

// AgentTerminalBroker manages per-agent WebSocket connections and handles
// runtime generation switches without remounting the browser xterm (Gate 4).
//
// Browser identity is AgentID. The broker observes the current
// RuntimeGeneration and reconnects the IPC transport underneath when the
// generation changes, emitting runtime_changed / agent_state /
// continuity_state frames to all observers.
type AgentTerminalBroker struct {
	mu     sync.RWMutex
	agents map[string]*agentTerminalState // agentID → state
}

type agentTerminalState struct {
	agentID        string
	runtimeID      string // current IPC target
	connections    map[*websocket.Conn]TerminalObserver
	writerConn     *websocket.Conn // single CONTROL writer
	lease          string          // "CONTROL" | "VIEW_ONLY"
	runtimeChanged chan struct{}   // closed + recreated on generation switch
}

// TerminalObserver represents one browser tab connected to an agent's terminal.
type TerminalObserver struct {
	conn    *websocket.Conn
	role    string // "CONTROL" | "VIEW_ONLY"
	created time.Time
}

var defaultBroker = &AgentTerminalBroker{
	agents: make(map[string]*agentTerminalState),
}

// DefaultBroker returns the global terminal broker singleton.
func DefaultBroker() *AgentTerminalBroker {
	return defaultBroker
}

// Attach registers a new WebSocket connection for the given agent. If the
// agent already has a runtime, it receives bounded history replay. Returns
// the assigned lease role.
func (b *AgentTerminalBroker) Attach(agentID string, conn *websocket.Conn, hasRuntime bool) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		state = &agentTerminalState{
			agentID:        agentID,
			connections:    make(map[*websocket.Conn]TerminalObserver),
			runtimeChanged: make(chan struct{}),
		}
		b.agents[agentID] = state
	}

	role := "VIEW_ONLY"
	if state.writerConn == nil && hasRuntime {
		role = "CONTROL"
		state.writerConn = conn
	}

	state.connections[conn] = TerminalObserver{
		conn:    conn,
		role:    role,
		created: time.Now(),
	}

	return role
}

// Detach removes a WebSocket connection and transfers writer lease if needed.
func (b *AgentTerminalBroker) Detach(agentID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		return
	}

	delete(state.connections, conn)

	// If the disconnected connection was the writer, transfer to next.
	if state.writerConn == conn {
		state.writerConn = nil
		for _, obs := range state.connections {
			state.writerConn = obs.conn
			state.lease = "CONTROL"
			// Notify new writer via out-of-band lease message.
			leaseMsg, _ := json.Marshal(map[string]string{
				"type": "lease",
				"role": "CONTROL",
			})
			_ = obs.conn.WriteMessage(websocket.TextMessage, leaseMsg)
			break
		}
	}

	// Clean up empty agent states.
	if len(state.connections) == 0 {
		delete(b.agents, agentID)
	}
}

// TakeControl steals the writer lease from the current writer.
func (b *AgentTerminalBroker) TakeControl(agentID string, conn *websocket.Conn) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		return false
	}

	if state.writerConn == conn {
		return true // already the writer
	}

	// Revoke old writer.
	if state.writerConn != nil {
		revokeMsg, _ := json.Marshal(map[string]string{
			"type": "lease",
			"role": "VIEW_ONLY",
		})
		_ = state.writerConn.WriteMessage(websocket.TextMessage, revokeMsg)
	}

	state.writerConn = conn
	state.lease = "CONTROL"
	return true
}

// ReleaseControl voluntarily releases the writer lease.
func (b *AgentTerminalBroker) ReleaseControl(agentID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok || state.writerConn != conn {
		return
	}

	state.writerConn = nil
	// Transfer to first available observer.
	for _, obs := range state.connections {
		state.writerConn = obs.conn
		leaseMsg, _ := json.Marshal(map[string]string{
			"type": "lease",
			"role": "CONTROL",
		})
		_ = obs.conn.WriteMessage(websocket.TextMessage, leaseMsg)
		break
	}
}

// IsWriter returns true if the given WebSocket connection currently holds the
// authoritative CONTROL lease for the agent (§45, A6 single writer authority).
func (b *AgentTerminalBroker) IsWriter(agentID string, conn *websocket.Conn) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return false
	}
	return state.writerConn == conn
}

// Writer returns the current authoritative writer connection for the agent.
func (b *AgentTerminalBroker) Writer(agentID string) *websocket.Conn {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return nil
	}
	return state.writerConn
}

// WatchRuntimeChanged returns a channel that is closed when the runtime
// generation changes for the given agent. Returns nil if agent has no state.
func (b *AgentTerminalBroker) WatchRuntimeChanged(agentID string) <-chan struct{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return nil
	}
	return state.runtimeChanged
}

// CurrentRuntimeID returns the current runtime ID for an agent, or empty string.
func (b *AgentTerminalBroker) CurrentRuntimeID(agentID string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return ""
	}
	return state.runtimeID
}

// NotifyRuntimeChanged broadcasts a runtime_changed frame to all observers
// of the given agent. Called by the Nexus layer when a generation switch occurs.
func (b *AgentTerminalBroker) NotifyRuntimeChanged(agentID string, oldRuntimeID, newRuntimeID, provider, profile, continuity string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		return
	}

	state.runtimeID = newRuntimeID

	if state.runtimeChanged != nil {
		close(state.runtimeChanged)
	}
	state.runtimeChanged = make(chan struct{})

	payload := protocol.RuntimeChangedPayload{
		AgentID:      agentID,
		OldRuntimeID: oldRuntimeID,
		NewRuntimeID: newRuntimeID,
		Provider:     provider,
		Profile:      profile,
		Continuity:   continuity,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "runtime_changed",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	for _, obs := range state.connections {
		_ = obs.conn.WriteMessage(websocket.TextMessage, msgBytes)
	}
}

// NotifyAgentState broadcasts agent state to all observers.
func (b *AgentTerminalBroker) NotifyAgentState(agentID, agentState string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ats, ok := b.agents[agentID]
	if !ok {
		return
	}

	payload := protocol.AgentStatePayload{
		AgentID: agentID,
		State:   agentState,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "agent_state",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	for _, obs := range ats.connections {
		_ = obs.conn.WriteMessage(websocket.TextMessage, msgBytes)
	}
}

// NotifyContinuity broadcasts continuity status to all observers.
func (b *AgentTerminalBroker) NotifyContinuity(agentID, continuity string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ats, ok := b.agents[agentID]
	if !ok {
		return
	}

	payload := protocol.ContinuityStatePayload{
		AgentID:    agentID,
		Continuity: continuity,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "continuity_state",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	for _, obs := range ats.connections {
		_ = obs.conn.WriteMessage(websocket.TextMessage, msgBytes)
	}
}

// ObserverCount returns the number of connected observers for an agent.
func (b *AgentTerminalBroker) ObserverCount(agentID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if state, ok := b.agents[agentID]; ok {
		return len(state.connections)
	}
	return 0
}
