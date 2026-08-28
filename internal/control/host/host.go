package host

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

// SessionHost manages a single supervised runtime process and coordinates IPC connections.
type SessionHost struct {
	mu           sync.RWMutex
	session      registry.RuntimeSession
	cmd          *exec.Cmd
	ptmx         *os.File
	stdinWriter  io.Writer
	ringBuffer   *RingBuffer
	listener     net.Listener
	clients      map[net.Conn]bool
	stopChan     chan struct{}
	stopped      bool
	onStateChange func(registry.RuntimeState)
}

// Config specifies options for launching a SessionHost.
type Config struct {
	Session  registry.RuntimeSession
	Binary   string
	Args     []string
	Env      []string
	Cwd      string
	UsePTY   bool
}

// NewSessionHost initializes a SessionHost for a given runtime.
func NewSessionHost(cfg Config) (*SessionHost, error) {
	if cfg.Session.RuntimeID == "" {
		return nil, errors.New("runtime ID is required")
	}

	cmd := exec.Command(cfg.Binary, cfg.Args...)
	cmd.Env = cfg.Env
	cmd.Dir = cfg.Cwd

	sh := &SessionHost{
		session:    cfg.Session,
		cmd:        cmd,
		ringBuffer: NewRingBuffer(128 * 1024), // 128 KB ring buffer
		clients:    make(map[net.Conn]bool),
		stopChan:   make(chan struct{}),
	}

	return sh, nil
}

// Start spawns the supervised child process and begins serving IPC requests.
func (sh *SessionHost) Start() error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// 1. Start child process with PTY if supported, otherwise standard pipes
	var err error
	var ptmx *os.File

	ptmx, err = pty.Start(sh.cmd)
	if err != nil {
		// Fallback to standard pipe execution if PTY fails
		inPipe, pipeErr := sh.cmd.StdinPipe()
		if pipeErr != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", pipeErr)
		}
		outPipe, pipeErr := sh.cmd.StdoutPipe()
		if pipeErr != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", pipeErr)
		}
		sh.cmd.Stderr = sh.cmd.Stdout

		if startErr := sh.cmd.Start(); startErr != nil {
			return fmt.Errorf("failed to start process: %w", startErr)
		}

		sh.stdinWriter = inPipe
		go sh.streamReader(outPipe)
	} else {
		sh.ptmx = ptmx
		sh.stdinWriter = ptmx
		go sh.streamReader(ptmx)
	}

	// Update session details
	sh.session.PID = sh.cmd.Process.Pid
	sh.session.State = registry.StateRunning
	sh.session.ControlEndpoint = protocol.EndpointPath(sh.session.RuntimeID)

	// Persist in Registry
	_ = registry.DefaultRegistry().Register(sh.session)

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
	reader := bufio.NewReader(conn)

	// Peek first byte to check if this is an interactive attach stream or command frame
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			// If not a JSON command, treat as raw input streaming
			sh.handleRawInput(line)
			continue
		}

		sh.handleRPCRequest(conn, req)
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
		// Send initial terminal history from ring buffer
		history := sh.ringBuffer.Bytes()
		resp, _ = protocol.NewResponse(string(history))

	case protocol.CmdDetach:
		delete(sh.clients, conn)
		resp, _ = protocol.NewResponse("detached")

	case protocol.CmdResize:
		if sh.ptmx != nil && req.Payload != nil {
			var p protocol.ResizePayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Rows > 0 && p.Cols > 0 {
				_ = pty.Setsize(sh.ptmx, &pty.Winsize{
					Rows: uint16(p.Rows),
					Cols: uint16(p.Cols),
				})
			}
		}
		resp, _ = protocol.NewResponse("resized")

	case protocol.CmdInput:
		if req.Payload != nil {
			var p protocol.InputPayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Data != "" {
				sh.handleRawInput([]byte(p.Data))
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

func (sh *SessionHost) handleRawInput(data []byte) {
	str := string(data)
	route := RouteSlashCommand(str, sh.session)

	if route.Intercepted {
		// Display response directly to attached clients without sending to child process
		if route.Response != "" {
			sh.broadcast([]byte(route.Response))
		}
		if route.Action == "detach" {
			// Detach clients
			sh.mu.Lock()
			for conn := range sh.clients {
				_ = conn.Close()
			}
			sh.clients = make(map[net.Conn]bool)
			sh.mu.Unlock()
		} else if route.Action == "stop" {
			go sh.Stop()
		}
		return
	}

	// Forward allowed/escaped input to child process stdin
	if route.ForwardToProcess != "" && sh.stdinWriter != nil {
		_, _ = sh.stdinWriter.Write([]byte(route.ForwardToProcess))
	}
}

func (sh *SessionHost) removeClient(conn net.Conn) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.clients, conn)
}

func (sh *SessionHost) waitProcess() {
	_ = sh.cmd.Wait()

	sh.mu.Lock()
	sh.session.State = registry.StateStopped
	_ = registry.DefaultRegistry().UpdateState(sh.session.RuntimeID, registry.StateStopped)
	sh.mu.Unlock()

	sh.Stop()
}

// Stop initiates graceful shutdown.
func (sh *SessionHost) Stop() {
	sh.mu.Lock()
	if sh.stopped {
		sh.mu.Unlock()
		return
	}
	sh.stopped = true
	close(sh.stopChan)

	if sh.listener != nil {
		_ = sh.listener.Close()
	}

	// Close all client connections
	for conn := range sh.clients {
		_ = conn.Close()
	}
	sh.clients = make(map[net.Conn]bool)

	// Send SIGTERM to process if still running
	if sh.cmd != nil && sh.cmd.Process != nil {
		_ = sh.cmd.Process.Signal(syscall.SIGTERM)
	}

	// Clean up socket file
	_ = os.Remove(protocol.EndpointPath(sh.session.RuntimeID))

	sh.mu.Unlock()
}

// Terminate sends SIGKILL.
func (sh *SessionHost) Terminate() {
	sh.Stop()
	if sh.cmd != nil && sh.cmd.Process != nil {
		_ = sh.cmd.Process.Kill()
	}
}
