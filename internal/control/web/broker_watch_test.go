package web

import (
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

func TestAttachWatchChannelIsOpen(t *testing.T) {
	b := &AgentTerminalBroker{agents: make(map[string]*agentTerminalState)}
	_ = b.Attach("agent-1", nil, true, &sync.Mutex{})
	ch := b.WatchRuntimeChanged("agent-1")
	if ch == nil {
		t.Fatal("attach must expose a runtime-changed channel")
	}
	select {
	case <-ch:
		t.Fatal("attach must not return an already-closed runtime-changed channel")
	default:
	}

	b.Detach("agent-1", nil)
	_ = b.Attach("agent-1", nil, true, &sync.Mutex{})
	ch2 := b.WatchRuntimeChanged("agent-1")
	if ch2 == nil {
		t.Fatal("reattach must expose a runtime-changed channel")
	}
	select {
	case <-ch2:
		t.Fatal("reattach must not inherit a closed runtime-changed channel")
	default:
	}
}

func TestAttachRecreatesNilRuntimeChanged(t *testing.T) {
	b := &AgentTerminalBroker{agents: make(map[string]*agentTerminalState)}
	b.agents["agent-1"] = &agentTerminalState{
		agentID:     "agent-1",
		connections: make(map[*websocket.Conn]TerminalObserver),
	}
	_ = b.Attach("agent-1", nil, true, &sync.Mutex{})
	ch := b.WatchRuntimeChanged("agent-1")
	if ch == nil {
		t.Fatal("attach must recreate a missing runtime-changed channel")
	}
	select {
	case <-ch:
		t.Fatal("recreated channel must be open")
	default:
	}
}
