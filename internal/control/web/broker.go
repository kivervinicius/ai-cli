package web

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/websocketio"
)

// AgentTerminalBroker manages per-agent WebSocket connections and handles
// runtime generation switches without remounting the browser xterm (Gate 4).
//
// Browser identity is AgentID. The broker observes the current
// RuntimeGeneration and reconnects the IPC transport underneath when the
// generation changes, emitting runtime_changed / agent_state /
// continuity_state frames to all observers.
//
// All writes to a browser connection MUST go through the observer's Writer.
// Gorilla WebSocket supports one concurrent reader and one concurrent writer;
// bypassing this gate reintroduces concurrent-writer corruption/panics.
type AgentTerminalBroker struct {
	mu     sync.RWMutex
	agents map[string]*agentTerminalState // agentID → state
}

type agentTerminalState struct {
	agentID        string
	runtimeID      string // current IPC target
	connections    map[*websocket.Conn]TerminalObserver
	writerConn     *websocket.Conn // Agent-level CONTROL owner
	runtimeChanged chan struct{}   // closed + recreated on generation switch
}

// TerminalObserver represents one browser tab connected to an agent's terminal.
type TerminalObserver struct {
	conn    *websocket.Conn
	writer  *websocketio.Writer
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

// Attach registers a new WebSocket connection for the given agent and returns
// both its Agent-level lease role and the one serialized writer that must be
// used for every server→browser frame on this connection.
func (b *AgentTerminalBroker) Attach(agentID string, conn *websocket.Conn, hasRuntime bool) (string, *websocketio.Writer) {
	writer := websocketio.NewWriter(conn.WriteJSON)

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
		writer:  writer,
		role:    role,
		created: time.Now(),
	}

	return role, writer
}

// Detach removes a WebSocket connection and transfers Agent-level CONTROL to
// one remaining observer. The promoted browser must acquire the SessionHost
// lease over its own attached IPC connection; AgentTerminal does that when it
// receives the CONTROL frame.
func (b *AgentTerminalBroker) Detach(agentID string, conn *websocket.Conn) {
	var promoted *websocketio.Writer

	b.mu.Lock()
	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return
	}

	delete(state.connections, conn)
	if state.writerConn == conn {
		state.writerConn = nil
		for candidate, obs := range state.connections {
			state.writerConn = candidate
			obs.role = "CONTROL"
			state.connections[candidate] = obs
			promoted = obs.writer
			break
		}
	}
	if len(state.connections) == 0 {
		delete(b.agents, agentID)
	}
	b.mu.Unlock()

	if promoted != nil {
		_ = promoted.WriteJSON(TerminalMessage{Type: "lease", Role: "CONTROL"})
	}
}

// TakeControl transfers Agent-level CONTROL to conn. SessionHost ownership is
// synchronized separately by CmdLeaseAcquire on conn's attached IPC stream.
func (b *AgentTerminalBroker) TakeControl(agentID string, conn *websocket.Conn) bool {
	var revoked *websocketio.Writer

	b.mu.Lock()
	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return false
	}
	if _, attached := state.connections[conn]; !attached {
		b.mu.Unlock()
		return false
	}
	if state.writerConn == conn {
		b.mu.Unlock()
		return true
	}

	if state.writerConn != nil {
		if old, exists := state.connections[state.writerConn]; exists {
			old.role = "VIEW_ONLY"
			state.connections[state.writerConn] = old
			revoked = old.writer
		}
	}
	current := state.connections[conn]
	current.role = "CONTROL"
	state.connections[conn] = current
	state.writerConn = conn
	b.mu.Unlock()

	if revoked != nil {
		_ = revoked.WriteJSON(TerminalMessage{Type: "lease", Role: "VIEW_ONLY"})
	}
	return true
}

// ReleaseControl voluntarily releases Agent-level CONTROL and promotes one
// remaining observer if available.
func (b *AgentTerminalBroker) ReleaseControl(agentID string, conn *websocket.Conn) {
	var promoted *websocketio.Writer

	b.mu.Lock()
	state, ok := b.agents[agentID]
	if !ok || state.writerConn != conn {
		b.mu.Unlock()
		return
	}

	if current, exists := state.connections[conn]; exists {
		current.role = "VIEW_ONLY"
		state.connections[conn] = current
	}
	state.writerConn = nil
	for candidate, obs := range state.connections {
		if candidate == conn {
			continue
		}
		state.writerConn = candidate
		obs.role = "CONTROL"
		state.connections[candidate] = obs
		promoted = obs.writer
		break
	}
	b.mu.Unlock()

	if promoted != nil {
		_ = promoted.WriteJSON(TerminalMessage{Type: "lease", Role: "CONTROL"})
	}
}

// IsWriter returns true if the given WebSocket connection currently holds the
// authoritative Agent-level CONTROL lease (§45, A6).
func (b *AgentTerminalBroker) IsWriter(agentID string, conn *websocket.Conn) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return false
	}
	return state.writerConn == conn
}

// Writer returns the current Agent-level writer connection for the agent.
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

func cloneObserverWriters(observers map[*websocket.Conn]TerminalObserver) []*websocketio.Writer {
	writers := make([]*websocketio.Writer, 0, len(observers))
	for _, obs := range observers {
		if obs.writer != nil {
			writers = append(writers, obs.writer)
		}
	}
	return writers
}

func broadcastTerminalMessage(writers []*websocketio.Writer, msg TerminalMessage) {
	for _, writer := range writers {
		_ = writer.WriteJSON(msg)
	}
}

// NotifyRuntimeChanged broadcasts a runtime_changed frame to all observers
// of the given agent. Called by the Nexus layer when a generation switch occurs.
func (b *AgentTerminalBroker) NotifyRuntimeChanged(agentID string, oldRuntimeID, newRuntimeID, provider, profile, continuity string) {
	b.mu.Lock()
	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return
	}

	state.runtimeID = newRuntimeID
	if state.runtimeChanged != nil {
		close(state.runtimeChanged)
	}
	state.runtimeChanged = make(chan struct{})
	writers := cloneObserverWriters(state.connections)
	b.mu.Unlock()

	payload := protocol.RuntimeChangedPayload{
		AgentID:      agentID,
		OldRuntimeID: oldRuntimeID,
		NewRuntimeID: newRuntimeID,
		Provider:     provider,
		Profile:      profile,
		Continuity:   continuity,
	}
	payloadBytes, _ := json.Marshal(payload)
	broadcastTerminalMessage(writers, TerminalMessage{Type: "runtime_changed", Data: string(payloadBytes)})
}

// NotifyAgentState broadcasts agent state to all observers.
func (b *AgentTerminalBroker) NotifyAgentState(agentID, agentState string) {
	b.mu.RLock()
	state, ok := b.agents[agentID]
	if !ok {
		b.mu.RUnlock()
		return
	}
	writers := cloneObserverWriters(state.connections)
	b.mu.RUnlock()

	payloadBytes, _ := json.Marshal(protocol.AgentStatePayload{AgentID: agentID, State: agentState})
	broadcastTerminalMessage(writers, TerminalMessage{Type: "agent_state", Data: string(payloadBytes)})
}

// NotifyContinuity broadcasts continuity status to all observers.
func (b *AgentTerminalBroker) NotifyContinuity(agentID, continuity string) {
	b.mu.RLock()
	state, ok := b.agents[agentID]
	if !ok {
		b.mu.RUnlock()
		return
	}
	writers := cloneObserverWriters(state.connections)
	b.mu.RUnlock()

	payloadBytes, _ := json.Marshal(protocol.ContinuityStatePayload{AgentID: agentID, Continuity: continuity})
	broadcastTerminalMessage(writers, TerminalMessage{Type: "continuity_state", Data: string(payloadBytes)})
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
