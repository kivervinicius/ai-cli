package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	mu                    sync.RWMutex
	session               registry.RuntimeSession
	cfg                   Config
	cmd                   *exec.Cmd
	termBackend           terminal.Backend
	ringBuffer            *RingBuffer
	fanout                *BoundedFanout
	listener              net.Listener
	clients               map[net.Conn]bool
	activeWriter          net.Conn
	pendingControlCommand string
	stopChan              chan struct{}
	doneChan              chan struct{}
	prefixRouter          *SlashPrefixRouter
	detector              *AttentionDetector
	stopOnce              sync.Once
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

	sh := &SessionHost{
		session:      cfg.Session,
		cfg:          cfg,
		cmd:          cmd,
		termBackend:  termBackend,
		ringBuffer:   NewRingBuffer(128 * 1024), // 128 KB terminal history
		fanout:       NewBoundedFanout(256),
		clients:      make(map[net.Conn]bool),
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
		prefixRouter: NewSlashPrefixRouter(),
	}

	sh.detector = NewAttentionDetectorWithProject(
		cfg.Session.RuntimeID,
		cfg.Session.ProviderID,
		cfg.Session.ProfileID,
		cfg.Cwd,
		cfg.Session.ProjectID,
		cfg.Session.ProjectName,
		func(reason, context, dynamicTitle string, state registry.RuntimeState) {
			sh.mu.Lock()
			sh.session.State = state
			sh.session.AttentionReason = reason
			sh.session.AttentionContext = context
			sh.session.DynamicTitle = dynamicTitle
			sh.session.Title = dynamicTitle
			if sh.detector != nil {
				sh.session.PromptKind = sh.detector.lastPrompt
				sh.session.AttentionKind = sh.detector.lastKind
				sh.session.AttentionFingerprint = sh.detector.lastFingerprint
			}
			sh.mu.Unlock()
		},
	)
	// PTY regex is TERMINAL fallback only. EVENTS/CONTROL_API (future structured
	// adapters) must not scrape stdout for agent attention.
	sh.detector.SetControlPolicy(cfg.Session.ControlLevel, false, cfg.Session.AgentID)

	return sh, nil
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

	prepareCmd(sh.cmd)

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
		// Clean up and terminate the spawned child process immediately so it is not orphaned
		_ = sh.termBackend.Kill()
		_ = sh.termBackend.Wait()
		_ = sh.termBackend.Close()
		sh.session.State = registry.StateFailed
		_ = registry.DefaultRegistry().UpdateState(sh.session.RuntimeID, registry.StateFailed)
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
	var pending []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				idx := bytes.IndexByte(pending, '\n')
				if idx < 0 {
					// Flush non-JSON partials so interactive output stays live.
					if len(pending) > 0 && pending[0] != '{' {
						chunk := pending
						pending = nil
						sh.ringBuffer.Write(chunk)
						sh.broadcast(chunk)
						if sh.detector != nil {
							sh.detector.ProcessChunk(chunk)
						}
					} else if len(pending) > MaxFrameSize {
						chunk := pending
						pending = nil
						sh.ringBuffer.Write(chunk)
						sh.broadcast(chunk)
						if sh.detector != nil {
							sh.detector.ProcessChunk(chunk)
						}
					}
					break
				}
				line := pending[:idx+1]
				pending = pending[idx+1:]
				if isHostProtocolControlFrame(line) {
					continue
				}
				sh.ringBuffer.Write(line)
				sh.broadcast(line)
				if sh.detector != nil {
					sh.detector.ProcessChunk(line)
				}
			}
		}
		if err != nil {
			if len(pending) > 0 && !isHostProtocolControlFrame(pending) {
				sh.ringBuffer.Write(pending)
				sh.broadcast(pending)
				if sh.detector != nil {
					sh.detector.ProcessChunk(pending)
				}
			}
			break
		}
	}
}

// isHostProtocolControlFrame detects Nexus Control RPC lines that must never
// enter the ring buffer / fanout (e.g. lease_acquire echoed into the PTY).
func isHostProtocolControlFrame(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(trimmed, &probe) != nil {
		return false
	}
	if _, hasCommand := probe["command"]; hasCommand {
		return true
	}
	if _, hasOK := probe["ok"]; hasOK {
		return true
	}
	return false
}

func (sh *SessionHost) broadcast(data []byte) {
	sh.fanout.Broadcast(data)
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
		line, err := readBoundedLine(reader, MaxFrameSize)
		if err != nil {
			if err == errFrameTooLarge {
				sh.broadcast([]byte("\r\n[Nexus Control] Error: oversized IPC frame rejected\r\n"))
			}
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

// errFrameTooLarge is returned when an IPC frame exceeds MaxFrameSize.
var errFrameTooLarge = errors.New("IPC frame exceeds maximum allowed size")

// readBoundedLine reads a newline-terminated frame while never allocating more
// than limit bytes. This prevents a malicious peer from forcing unbounded
// memory growth by omitting the newline (see readBytes-safety requirement).
func readBoundedLine(r *bufio.Reader, limit int) ([]byte, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return buf, err
		}
		if b == '\n' {
			return buf, nil
		}
		buf = append(buf, b)
		if len(buf) > limit {
			return buf, errFrameTooLarge
		}
	}
}

func (sh *SessionHost) streamAttachedInput(conn net.Conn, reader *bufio.Reader) {
	_ = conn.SetDeadline(time.Time{})
	// Mix of newline-terminated RPC frames and raw PTY keystrokes on one
	// stream. Raw bytes must be forwarded immediately — interactive shells
	// send single keystrokes with no trailing \n (Enter is usually \r).
	var jsonBuf []byte
	inJSON := false

	flushRaw := func(data []byte) {
		if len(data) == 0 {
			return
		}
		sh.processAttachedInput(conn, data)
	}
	resetJSON := func() {
		jsonBuf = nil
		inJSON = false
	}
	looksLikeProtocol := func(buf []byte) bool {
		return bytes.HasPrefix(buf, []byte(`{"version"`)) ||
			bytes.HasPrefix(buf, []byte(`{"command"`)) ||
			bytes.HasPrefix(buf, []byte(`{"id"`))
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if inJSON && len(jsonBuf) > 0 {
				flushRaw(jsonBuf)
			}
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		if !inJSON {
			if b == '{' {
				inJSON = true
				jsonBuf = append(jsonBuf[:0], b)
				continue
			}
			flushRaw([]byte{b})
			continue
		}

		jsonBuf = append(jsonBuf, b)
		if len(jsonBuf) > MaxFrameSize {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		// Enter (\r) mid-buffer means interactive typing, not an RPC frame.
		if b == '\r' {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		// Once long enough, reject buffers that cannot be protocol JSON.
		if len(jsonBuf) >= 11 && !looksLikeProtocol(jsonBuf) {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		if b != '\n' {
			continue
		}

		trimmed := bytes.TrimSpace(jsonBuf)
		var req protocol.Request
		if json.Unmarshal(trimmed, &req) == nil && req.Command != "" {
			sh.handleRPCRequest(conn, req)
		} else {
			flushRaw(jsonBuf)
		}
		resetJSON()
	}
}

func (sh *SessionHost) handleRPCRequest(conn net.Conn, req protocol.Request) {
	sh.mu.Lock()

	if req.Version != 0 && req.Version != protocol.ProtocolVersion {
		resp := protocol.NewErrorResponse("ERROR_PROTOCOL_VERSION")
		sh.mu.Unlock()
		data, _ := json.Marshal(resp)
		_, _ = conn.Write(append(data, '\n'))
		return
	}

	var resp protocol.Response
	var resizeRows, resizeCols int
	var promptWrite []byte
	var forwardWrite []byte
	var controlCmd string
	skipResponse := false

	switch req.Command {
	case protocol.CmdPing:
		resp, _ = protocol.NewResponse("pong")

	case protocol.CmdStatus:
		fanoutStats := sh.fanout.Stats()
		resp, _ = protocol.NewResponse(protocol.StatusData{
			RuntimeID:           sh.session.RuntimeID,
			ProviderID:          sh.session.ProviderID,
			ProfileID:           sh.session.ProfileID,
			ProviderSessionID:   sh.session.ProviderSessionID,
			Workspace:           sh.session.Workspace,
			PID:                 sh.session.PID,
			State:               string(sh.session.State),
			ControlLevel:        string(sh.session.ControlLevel),
			StartedAt:           sh.session.StartedAt,
			DroppedOutputChunks: fanoutStats.DroppedChunks,
		})

	case protocol.CmdAttach:
		sh.clients[conn] = true
		sh.fanout.AddClient(conn)
		if sh.activeWriter == nil {
			sh.activeWriter = conn
		}
		history := sh.ringBuffer.Bytes()
		resp, _ = protocol.NewResponse(string(history))

	case protocol.CmdDetach:
		delete(sh.clients, conn)
		sh.fanout.RemoveClient(conn)
		if sh.activeWriter == conn {
			sh.activeWriter = nil
		}
		resp, _ = protocol.NewResponse("detached")

	case protocol.CmdResize:
		if req.Payload != nil {
			var p protocol.ResizePayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Rows > 0 && p.Cols > 0 {
				resizeRows, resizeCols = p.Rows, p.Cols
			}
		}
		resp, _ = protocol.NewResponse("resized")

	case protocol.CmdInput:
		if req.Payload != nil {
			var p protocol.InputPayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Data != "" {
				forwardWrite = sh.collectAttachedInputLocked(conn, []byte(p.Data))
				controlCmd = sh.pendingControlCommand
				sh.pendingControlCommand = ""
			}
		}
		resp, _ = protocol.NewResponse("input_received")

	case protocol.CmdSubmitPrompt:
		var p protocol.SubmitPromptPayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || strings.TrimSpace(p.Prompt) == "" {
			resp = protocol.NewErrorResponse("prompt is required")
			break
		}
		prompt := strings.TrimRight(p.Prompt, "\r\n") + "\r"
		promptWrite = []byte(prompt)
		resp, _ = protocol.NewResponse("prompt_submitted")

	case protocol.CmdStop:
		go sh.Stop()
		resp, _ = protocol.NewResponse("stopping")

	case protocol.CmdTerminate:
		go sh.Terminate()
		resp, _ = protocol.NewResponse("terminated")

	case protocol.CmdLeaseAcquire:
		if _, attached := sh.clients[conn]; !attached {
			resp = protocol.NewErrorResponse("lease acquisition requires attached streaming client")
			break
		}
		sh.activeWriter = conn
		resp, _ = protocol.NewResponse("lease_acquired")

	case protocol.CmdLeaseRelease:
		if sh.activeWriter == conn {
			sh.activeWriter = nil
		}
		resp, _ = protocol.NewResponse("lease_released")

	default:
		resp = protocol.NewErrorResponse(fmt.Sprintf("unknown command %q", req.Command))
	}

	_, isAttached := sh.clients[conn]
	if isAttached && req.Command == protocol.CmdResize {
		skipResponse = true
	}
	sh.mu.Unlock()

	if resizeRows > 0 && resizeCols > 0 {
		_ = sh.termBackend.Resize(resizeRows, resizeCols)
	}
	if len(forwardWrite) > 0 {
		_, _ = sh.termBackend.Write(forwardWrite)
	}
	if len(promptWrite) > 0 {
		_, _ = sh.termBackend.Write(promptWrite)
	}
	if controlCmd != "" {
		sh.mu.Lock()
		sh.handleControlCommandLocked(controlCmd)
		sh.mu.Unlock()
	}
	if skipResponse {
		return
	}
	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

func (sh *SessionHost) processAttachedInput(conn net.Conn, data []byte) {
	sh.mu.Lock()
	forward := sh.collectAttachedInputLocked(conn, data)
	controlCmd := sh.pendingControlCommand
	sh.pendingControlCommand = ""
	sh.mu.Unlock()

	if len(forward) > 0 {
		_, _ = sh.termBackend.Write(forward)
	}
	if controlCmd != "" {
		sh.mu.Lock()
		sh.handleControlCommandLocked(controlCmd)
		sh.mu.Unlock()
	}
}

func (sh *SessionHost) collectAttachedInputLocked(conn net.Conn, data []byte) []byte {
	var forward []byte
	// Only active writer (or first attached client) can send input to child process
	if sh.activeWriter != nil && sh.activeWriter != conn {
		return nil
	}
	if sh.activeWriter == nil {
		sh.activeWriter = conn
	}

	if bytes.Equal(data, []byte("\x1b[I")) || bytes.Equal(data, []byte("\x1b[O")) ||
		bytes.Equal(data, []byte("[I")) || bytes.Equal(data, []byte("[O")) ||
		bytes.Equal(data, []byte("\x1b[1;1R")) || bytes.Equal(data, []byte("[1;1R")) {
		return nil
	}

	for _, b := range data {
		out := sh.prefixRouter.ProcessByte(b)
		if len(out.ForwardBytes) > 0 {
			forward = append(forward, out.ForwardBytes...)
		}
		if out.Action == ActionControlCommand && out.ControlCmd != "" {
			sh.pendingControlCommand = out.ControlCmd
		}
	}
	return forward
}

func (sh *SessionHost) handleControlCommandLocked(cmd string) {
	route := RouteSlashCommand(cmd, sh.session)
	if route.Intercepted && route.Response != "" {
		sh.broadcast([]byte("\r\n" + route.Response + "\r\n"))
	}

	switch route.Action {
	case "detach":
		for c := range sh.clients {
			_ = c.Close()
		}
		sh.clients = make(map[net.Conn]bool)
		sh.fanout.Close()
		sh.activeWriter = nil
	case "stop":
		go sh.Stop()
	case "handoff":
		if PerformAccountHandoff != nil {
			go func(target string) {
				_, err := PerformAccountHandoff(context.Background(), sh.session.RuntimeID, target)
				if err != nil {
					sh.broadcast([]byte(fmt.Sprintf("\r\n[Nexus Control] Handoff failed: %v\r\n", err)))
				} else {
					sh.broadcast([]byte("\r\n[Nexus Control] Handoff succeeded.\r\n"))
				}
			}(route.ActionArg)
		}
	case "continue":
		if PerformContextHandoff != nil {
			go func(target string) {
				var provider, profile string
				if idx := strings.Index(target, ":"); idx != -1 {
					provider = target[:idx]
					profile = target[idx+1:]
				} else {
					provider = target
				}
				_, err := PerformContextHandoff(context.Background(), sh.session.RuntimeID, provider, profile)
				if err != nil {
					sh.broadcast([]byte(fmt.Sprintf("\r\n[Nexus Control] Context handoff failed: %v\r\n", err)))
				} else {
					sh.broadcast([]byte("\r\n[Nexus Control] Context handoff succeeded.\r\n"))
				}
			}(route.ActionArg)
		}
	}
}

func (sh *SessionHost) removeClient(conn net.Conn) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.clients, conn)
	sh.fanout.RemoveClient(conn)
	if sh.activeWriter == conn {
		sh.activeWriter = nil
	}
}

func (sh *SessionHost) waitProcess() {
	err := sh.termBackend.Wait()

	sh.mu.Lock()
	defer sh.mu.Unlock()

	_ = sh.termBackend.Close()

	// Tear down IPC surfaces so attached clients disconnect cleanly instead of
	// hanging on a dead runtime.
	if sh.listener != nil {
		_ = sh.listener.Close()
	}
	sh.fanout.Close()
	for c := range sh.clients {
		_ = c.Close()
	}
	sh.clients = make(map[net.Conn]bool)
	sh.activeWriter = nil

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
	_ = sh.termBackend.Signal(os.Interrupt)
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

	_ = sh.termBackend.Kill()
	if sh.listener != nil {
		_ = sh.listener.Close()
	}
	sh.fanout.Close()
	_ = sh.termBackend.Close()
	return nil
}
