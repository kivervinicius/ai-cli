package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return strings.HasPrefix(origin, "http://127.0.0.1") ||
			strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "http://[::1]")
	},
}

type TerminalMessage struct {
	Type string `json:"type"` // "output", "input", "resize", "lease", "error"
	Data string `json:"data,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Role string `json:"role,omitempty"` // "CONTROL" or "VIEW_ONLY"
}

type TerminalHub struct {
	mu     sync.Mutex
	leases map[string]*websocket.Conn // runtimeID -> active writer conn
}

func NewTerminalHub() *TerminalHub {
	return &TerminalHub{
		leases: make(map[string]*websocket.Conn),
	}
}

func (h *TerminalHub) HandleWebSocket(w http.ResponseWriter, r *http.Request, runtimeID string) {
	reg := registry.DefaultRegistry()
	_, exists := reg.Get(runtimeID)
	if !exists {
		http.Error(w, "runtime not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// Connect to runtime SessionHost via local IPC
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		_ = reg.UpdateState(runtimeID, registry.StateStopped)
		_ = ws.WriteJSON(TerminalMessage{
			Type: "error",
			Data: "Runtime host is not running (" + err.Error() + "). The process has exited or the socket was closed.",
		})
		return
	}
	defer client.Close()

	// Attach to host
	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		_ = reg.UpdateState(runtimeID, registry.StateStopped)
		_ = ws.WriteJSON(TerminalMessage{
			Type: "error",
			Data: "Failed to attach: runtime host is no longer responding (" + err.Error() + ").",
		})
		return
	}
	_ = client.ClearDeadline()

	// Initial ring buffer history
	var history string
	if json.Unmarshal(resp.Data, &history) == nil && history != "" {
		_ = ws.WriteJSON(TerminalMessage{
			Type: "output",
			Data: history,
		})
	}

	// Lease allocation
	h.mu.Lock()
	hasLease := false
	if h.leases[runtimeID] == nil {
		h.leases[runtimeID] = ws
		hasLease = true
	}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		if h.leases[runtimeID] == ws {
			delete(h.leases, runtimeID)
		}
		h.mu.Unlock()
	}()

	role := "VIEW_ONLY"
	if hasLease {
		role = "CONTROL"
	}
	_ = ws.WriteJSON(TerminalMessage{
		Type: "lease",
		Role: role,
	})

	rawConn := client.RawConn()
	stopChan := make(chan struct{})

	// 1. Pump stdout from child process to browser
	go func() {
		buf := make([]byte, 2048)
		for {
			select {
			case <-stopChan:
				return
			default:
				_ = rawConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, err := rawConn.Read(buf)
				if n > 0 {
					_ = ws.WriteJSON(TerminalMessage{
						Type: "output",
						Data: string(buf[:n]),
					})
				}
				if err != nil && !strings.Contains(err.Error(), "timeout") {
					return
				}
			}
		}
	}()

	// 2. Read WebSocket frames from browser
	for {
		var msg TerminalMessage
		if err := ws.ReadJSON(&msg); err != nil {
			close(stopChan)
			return
		}

		switch msg.Type {
		case "input":
			h.mu.Lock()
			currentWriter := h.leases[runtimeID]
			h.mu.Unlock()

			if currentWriter == ws && msg.Data != "" {
				_, _ = rawConn.Write([]byte(msg.Data))
			}

		case "resize":
			if msg.Rows > 0 && msg.Cols > 0 {
				_ = client.Resize(msg.Rows, msg.Cols)
			}

		case "lease_acquire":
			h.mu.Lock()
			h.leases[runtimeID] = ws
			h.mu.Unlock()
			_ = ws.WriteJSON(TerminalMessage{
				Type: "lease",
				Role: "CONTROL",
			})

		case "lease_release":
			h.mu.Lock()
			if h.leases[runtimeID] == ws {
				delete(h.leases, runtimeID)
			}
			h.mu.Unlock()
			_ = ws.WriteJSON(TerminalMessage{
				Type: "lease",
				Role: "VIEW_ONLY",
			})
		}
	}
}
