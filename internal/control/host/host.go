package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/control/terminal"
)

const MaxFrameSize = 64 * 1024 // 64 KB max frame

// Config configures a SessionHost instance.
type Config struct {
	Session     registry.RuntimeSession
	Binary      string
	Args        []string
	Env         []string
	Cwd         string
	InitialRows int
	InitialCols int
}

// SessionHost manages a single supervised process runtime and its IPC listener.
type SessionHost struct {
	mu           sync.RWMutex
	session      registry.RuntimeSession
	cfg          Config
	cmd          *exec.Cmd
	termBackend  terminal.Backend
	ringBuffer   *RingBuffer
	listener     net.Listener
	clients      map[net.Conn]bool
	activeWriter net.Conn
	stopChan     chan struct{}
	doneChan     chan struct{}
	lineBuf      bytes.Buffer
	stopOnce     sync.Once
}

// NewSessionHost creates a new SessionHost for a given runtime.
func NewSessionHost(cfg Config) (*SessionHost, error) {
	if cfg.Session.RuntimeID == "" {
		return nil, fmt.Errorf("runtime ID is required")
	}

	cmd := exec.Command(cfg.Binary, cfg.Args...)
	cmd.Dir = cfg.Cwd
	cmd.Env = cfg.Env

	termBackend := terminal.NewBackend()

	return &SessionHost{
		session:     cfg.Session,
		cfg:         cfg,
		cmd:         cmd,
		termBackend: termBackend,
		ringBuffer:  NewRingBuffer(128 * 1024), // 128 KB terminal history
		clients:     make(map[net.Conn]bool),
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}, nil
}

// Start launches the supervised process and begins listening for IPC control connections.
func (sh *SessionHost) Start() error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// 1. Start process with terminal backend
	rows := sh.cfg.InitialRows
	cols := sh.cfg.InitialCols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	if err := sh.termBackend.Start(sh.cmd, rows, cols); err != nil {
		return fmt.Errorf("failed to start terminal backend: %w", err)
	}

	// Start reading stdout from terminal backend
	go sh.streamReader(sh.termBackend)

	// Update session details
	sh.session.PID = sh.termBackend.PID()
	sh.session.HostPID = os.Getpid()
	sh.session.HostGeneration = time.Now().UnixNano()
	sh.session.State = registry.StateRunning
	sh.session.ControlEndpoint = protocol.EndpointPath(sh.session.RuntimeID)

	// Persist in Registry
	_ = registry.DefaultRegistry().Register(sh.session)

	// Emit Process Started event
	events.DefaultBus().Publish(events.NewEvent(
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventProcessStarted,
		fmt.Sprintf("Started supervised %s runtime (PID %d, Host PID %d)", sh.session.ProviderID, sh.session.PID, sh.session.HostPID),
		map[string]any{"pid": sh.session.PID, "host_pid": sh.session.HostPID, "endpoint": sh.session.ControlEndpoint},
	))

	// 2. Start IPC listener
	l, err := protocol.Listen(sh.session.RuntimeID)
	if err != nil {
		return fmt.Errorf("failed to create control endpoint: %w", err)
	}
	sh.listener = l
	go sh.serveIPC()

	// 3. Monitor process termination
	go sh.waitProcess()

	return nil
}

func (sh *SessionHost) streamReader(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			sh.ringBuffer.Write(chunk)
			sh.broadcast(chunk)
		}
		if err != nil {
			break
		}
	}
}

func (sh *SessionHost) broadcast(data []byte) {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	for conn := range sh.clients {
		_, _ = conn.Write(data)
	}
}

func (sh *SessionHost) serveIPC() {
	for {
		conn, err := sh.listener.Accept()
		if err != nil {
			select {
			case <-sh.stopChan:
				return
			default:
				return
			}
		}
		go sh.handleClient(conn)
	}
}

func (sh *SessionHost) handleClient(conn net.Conn) {
	reader := bufio.NewReaderSize(conn, MaxFrameSize)

	// Read initial RPC command frames
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		if len(line) > MaxFrameSize {
			_ = conn.Close()
			sh.removeClient(conn)
			return
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			// If not a JSON command, process raw input if writer
			sh.processAttachedInput(conn, line)
			continue
		}

		// Handle RPC request
		isAttach := req.Command == protocol.CmdAttach
		sh.handleRPCRequest(conn, req)

		// If this was an Attach command, switch connection to continuous raw streaming mode
		if isAttach {
			_ = conn.SetDeadline(time.Time{})
			sh.streamAttachedInput(conn, reader)
			return
		}
	}
}

func (sh *SessionHost) streamAttachedInput(conn net.Conn, reader *bufio.Reader) {
	_ = conn.SetDeadline(time.Time{})
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			sh.processAttachedInput(conn, buf[:n])
		}
		if err != nil {
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}
	}
}

func (sh *SessionHost) handleRPCRequest(conn net.Conn, req protocol.Request) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var resp protocol.Response

	switch req.Command {
	case protocol.CmdPing:
		resp, _ = protocol.NewResponse("pong")

	case protocol.CmdStatus:
		resp, _ = protocol.NewResponse(protocol.StatusData{
			RuntimeID:         sh.session.RuntimeID,
			ProviderID:        sh.session.ProviderID,
			ProfileID:         sh.session.ProfileID,
			ProviderSessionID: sh.session.ProviderSessionID,
			Workspace:         sh.session.Workspace,
			PID:               sh.session.PID,
			State:             string(sh.session.State),
			ControlLevel:      string(sh.session.ControlLevel),
			StartedAt:         sh.session.StartedAt,
		})

	case protocol.CmdAttach:
		sh.clients[conn] = true
		// Acquire single-writer lease if none active
		if sh.activeWriter == nil {
			sh.activeWriter = conn
		}
		// Send initial terminal history from ring buffer
		history := sh.ringBuffer.Bytes()
		resp, _ = protocol.NewResponse(string(history))

	case protocol.CmdDetach:
		delete(sh.clients, conn)
		if sh.activeWriter == conn {
			sh.activeWriter = nil
		}
		resp, _ = protocol.NewResponse("detached")

	case protocol.CmdResize:
		if req.Payload != nil {
			var p protocol.ResizePayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Rows > 0 && p.Cols > 0 {
				_ = sh.termBackend.Resize(p.Rows, p.Cols)
			}
		}
		resp, _ = protocol.NewResponse("resized")

	case protocol.CmdInput:
		if req.Payload != nil {
			var p protocol.InputPayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Data != "" {
				sh.processAttachedInput(conn, []byte(p.Data))
			}
		}
		resp, _ = protocol.NewResponse("input_received")

	case protocol.CmdStop:
		go sh.Stop()
		resp, _ = protocol.NewResponse("stopping")

	case protocol.CmdTerminate:
		go sh.Terminate()
		resp, _ = protocol.NewResponse("terminated")

	default:
		resp = protocol.NewErrorResponse(fmt.Sprintf("unknown command %q", req.Command))
	}

	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

func (sh *SessionHost) processAttachedInput(conn net.Conn, data []byte) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Only active writer (or first attached client) can send input to child process
	if sh.activeWriter != nil && sh.activeWriter != conn {
		return
	}
	if sh.activeWriter == nil {
		sh.activeWriter = conn
	}

	for _, b := range data {
		if b == '\r' || b == '\n' {
			line := sh.lineBuf.String()
			sh.lineBuf.Reset()

			clean := StripANSI(line)
			trimmed := strings.TrimSpace(clean)

			// 1. Check for escape prefix "//ai ..."
			if strings.HasPrefix(trimmed, "//ai") {
				// Clear the line from the child process readline buffer with Ctrl+U (0x15)
				_, _ = sh.termBackend.Write([]byte{0x15})
				escaped := "/ai" + trimmed[4:] + "\r"
				_, _ = sh.termBackend.Write([]byte(escaped))
				continue
			}

			// 2. Check for reserved "/ai" commands
			if strings.HasPrefix(trimmed, "/ai") {
				// Wipe the typed /ai command from the child process readline buffer with Ctrl+U (0x15)
				_, _ = sh.termBackend.Write([]byte{0x15})

				route := RouteSlashCommand(trimmed, sh.session)
				if route.Intercepted && route.Response != "" {
					sh.broadcast([]byte("\r\n" + route.Response + "\r\n"))
				}

				switch route.Action {
				case "detach":
					for c := range sh.clients {
						_ = c.Close()
					}
					sh.clients = make(map[net.Conn]bool)
					sh.activeWriter = nil
				case "stop":
					go sh.Stop()
				case "handoff":
					if PerformAccountHandoff != nil {
						go func(target string) {
							_, err := PerformAccountHandoff(context.Background(), sh.session.RuntimeID, target)
							if err != nil {
								sh.broadcast([]byte(fmt.Sprintf("\r\n[AI Control] Handoff failed: %v\r\n", err)))
							} else {
								sh.broadcast([]byte("\r\n[AI Control] Handoff succeeded.\r\n"))
								// The old process will be quiesced and clients disconnected by PerformAccountHandoff via protocol.Stop()
							}
						}(route.ActionArg)
					}
				case "continue":
					if PerformContextHandoff != nil {
						go func(target string) {
							// target is provider[:profile]
							var provider, profile string
							if idx := strings.Index(target, ":"); idx != -1 {
								provider = target[:idx]
								profile = target[idx+1:]
							} else {
								provider = target
							}
							_, err := PerformContextHandoff(context.Background(), sh.session.RuntimeID, provider, profile)
							if err != nil {
								sh.broadcast([]byte(fmt.Sprintf("\r\n[AI Control] Continue failed: %v\r\n", err)))
							} else {
								sh.broadcast([]byte("\r\n[AI Control] Continue succeeded.\r\n"))
							}
						}(route.ActionArg)
					}
				}
				continue
			}

			// 3. Normal line submission to child process
			_, _ = sh.termBackend.Write([]byte{'\r'})
		} else if b == 0x03 || b == 0x15 { // Ctrl+C or Ctrl+U
			sh.lineBuf.Reset()
			_, _ = sh.termBackend.Write([]byte{b})
		} else if b == 0x7f || b == 0x08 { // Backspace
			if sh.lineBuf.Len() > 0 {
				buf := sh.lineBuf.Bytes()
				sh.lineBuf.Reset()
				sh.lineBuf.Write(buf[:len(buf)-1])
			}
			_, _ = sh.termBackend.Write([]byte{b})
		} else {
			sh.lineBuf.WriteByte(b)
			_, _ = sh.termBackend.Write([]byte{b})
		}
	}
}

func (sh *SessionHost) removeClient(conn net.Conn) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.clients, conn)
	if sh.activeWriter == conn {
		sh.activeWriter = nil
	}
}

func (sh *SessionHost) waitProcess() {
	err := sh.cmd.Wait()

	sh.mu.Lock()
	defer sh.mu.Unlock()

	_ = sh.termBackend.Close()

	state := registry.StateStopped
	if err != nil {
		state = registry.StateFailed
	}
	sh.session.State = state
	_ = registry.DefaultRegistry().UpdateState(sh.session.RuntimeID, state)

	events.DefaultBus().Publish(events.NewEvent(
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventProcessExited,
		fmt.Sprintf("Process exited (State: %s)", state),
		map[string]any{"state": string(state)},
	))

	close(sh.doneChan)
}

// Wait blocks until the supervised child process terminates.
func (sh *SessionHost) Wait() {
	<-sh.doneChan
}

// Stop gracefully stops the supervised process.
func (sh *SessionHost) Stop() error {
	sh.stopOnce.Do(func() { close(sh.stopChan) })
	sh.mu.Lock()
	if sh.cmd.Process != nil {
		_ = sh.cmd.Process.Signal(os.Interrupt)
	}
	sh.mu.Unlock()

	select {
	case <-sh.doneChan:
		return nil
	case <-time.After(3 * time.Second):
		return sh.Terminate()
	}
}

// Terminate forcefully kills the supervised process.
func (sh *SessionHost) Terminate() error {
	sh.stopOnce.Do(func() { close(sh.stopChan) })
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.cmd.Process != nil {
		_ = sh.cmd.Process.Kill()
	}
	if sh.listener != nil {
		_ = sh.listener.Close()
	}
	_ = sh.termBackend.Close()
	return nil
}
