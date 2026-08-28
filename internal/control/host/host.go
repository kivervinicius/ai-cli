package host

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"golang.org/x/term"
)

// Config configures a SessionHost instance.
type Config struct {
	Session registry.RuntimeSession
	Binary  string
	Args    []string
	Env     []string
	Cwd     string
	UsePTY  bool
}

// SessionHost manages a single supervised process runtime and its IPC listener.
type SessionHost struct {
	mu          sync.RWMutex
	session     registry.RuntimeSession
	cfg         Config
	cmd         *exec.Cmd
	ptmx        *os.File
	stdinWriter io.Writer
	ringBuffer  *RingBuffer
	listener    net.Listener
	clients     map[net.Conn]bool
	stopChan    chan struct{}
	doneChan    chan struct{}
	lineBuf     bytes.Buffer
}

// NewSessionHost creates a new SessionHost for a given runtime.
func NewSessionHost(cfg Config) (*SessionHost, error) {
	if cfg.Session.RuntimeID == "" {
		return nil, fmt.Errorf("runtime ID is required")
	}

	cmd := exec.Command(cfg.Binary, cfg.Args...)
	cmd.Dir = cfg.Cwd
	cmd.Env = cfg.Env

	return &SessionHost{
		session:    cfg.Session,
		cfg:        cfg,
		cmd:        cmd,
		ringBuffer: NewRingBuffer(128 * 1024), // 128 KB terminal history
		clients:    make(map[net.Conn]bool),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}, nil
}

// Start launches the supervised process and begins listening for IPC control connections.
func (sh *SessionHost) Start() error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// 1. Start child process with PTY if supported, otherwise standard pipes
	var err error
	var ptmx *os.File

	var initialSize *pty.Winsize
	if w, h, sizeErr := term.GetSize(int(os.Stdout.Fd())); sizeErr == nil && w > 0 && h > 0 {
		initialSize = &pty.Winsize{Rows: uint16(h), Cols: uint16(w)}
	} else {
		initialSize = &pty.Winsize{Rows: 24, Cols: 80}
	}

	ptmx, err = pty.StartWithSize(sh.cmd, initialSize)
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

	// Emit Process Started event
	events.DefaultBus().Publish(events.NewEvent(
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventProcessStarted,
		fmt.Sprintf("Started supervised %s runtime (PID %d)", sh.session.ProviderID, sh.session.PID),
		map[string]any{"pid": sh.session.PID, "endpoint": sh.session.ControlEndpoint},
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
	reader := bufio.NewReader(conn)

	// Read initial RPC command frames
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			// If not a JSON command, process raw input
			sh.processAttachedInput(line)
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
		// Read raw bytes from terminal client
		n, err := reader.Read(buf)
		if n > 0 {
			sh.processAttachedInput(buf[:n])
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
				sh.processAttachedInput([]byte(p.Data))
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

func (sh *SessionHost) processAttachedInput(data []byte) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	for _, b := range data {
		if b == '\r' || b == '\n' {
			line := sh.lineBuf.String()
			sh.lineBuf.Reset()

			route := RouteSlashCommand(line, sh.session)
			if route.Intercepted {
				if route.Response != "" {
					sh.broadcast([]byte("\r\n" + route.Response + "\r\n"))
				}
				switch route.Action {
				case "detach":
					for conn := range sh.clients {
						_ = conn.Close()
					}
					sh.clients = make(map[net.Conn]bool)
				case "stop":
					go sh.Stop()
				}
				continue
			}

			// If it was an escaped command "//ai ...", forward unescaped "/ai ...\r" to process
			if strings.HasPrefix(line, "//ai") {
				if sh.stdinWriter != nil {
					_, _ = sh.stdinWriter.Write([]byte(route.ForwardToProcess + "\r"))
				}
				continue
			}

			// Normal line: send Enter to process
			if sh.stdinWriter != nil {
				_, _ = sh.stdinWriter.Write([]byte{'\r'})
			}
		} else if b == 0x7f || b == 0x08 { // Backspace
			if sh.lineBuf.Len() > 0 {
				buf := sh.lineBuf.Bytes()
				sh.lineBuf.Reset()
				sh.lineBuf.Write(buf[:len(buf)-1])
			}
			if sh.stdinWriter != nil {
				_, _ = sh.stdinWriter.Write([]byte{b})
			}
		} else {
			sh.lineBuf.WriteByte(b)
			curr := sh.lineBuf.String()

			// Check if we are potentially typing a slash command prefix
			isSlashPrefix := strings.HasPrefix("/ai", curr) ||
				strings.HasPrefix("//ai", curr) ||
				strings.HasPrefix(curr, "/ai") ||
				strings.HasPrefix(curr, "//ai")

			if !isSlashPrefix {
				if sh.stdinWriter != nil {
					_, _ = sh.stdinWriter.Write([]byte{b})
				}
			} else {
				// Echo slash command character locally to attached clients
				sh.broadcast([]byte{b})
			}
		}
	}
}

func (sh *SessionHost) removeClient(conn net.Conn) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.clients, conn)
}

func (sh *SessionHost) waitProcess() {
	err := sh.cmd.Wait()

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.ptmx != nil {
		_ = sh.ptmx.Close()
	}

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

// Stop gracefully stops the supervised process.
func (sh *SessionHost) Stop() error {
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
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.cmd.Process != nil {
		_ = sh.cmd.Process.Kill()
	}
	if sh.listener != nil {
		_ = sh.listener.Close()
	}
	return nil
}
