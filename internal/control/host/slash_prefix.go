package host

import (
	"bytes"
	"strings"
)

type PrefixState int

const (
	StateIdle PrefixState = iota
	StateSlash
	StateSlashA
	StateSlashAI
	StateControlCommand
	StateEscapeSlash
	StateEscapeSlashA
	StateEscapeSlashAI
	StatePassthrough
)

type RouterActionType int

const (
	ActionNone RouterActionType = iota
	ActionForwardBytes
	ActionControlCommand
)

type RouterOutput struct {
	Action       RouterActionType
	ForwardBytes []byte
	ControlCmd   string
}

// SlashPrefixRouter handles input bytes character by character, buffering only ambiguous slash prefixes
// (/ or /a or /ai) and instantly forwarding non-matching bytes to the child process.
type SlashPrefixRouter struct {
	state      PrefixState
	prefixBuf  bytes.Buffer
	commandBuf bytes.Buffer
}

// NewSlashPrefixRouter creates an initialized SlashPrefixRouter.
func NewSlashPrefixRouter() *SlashPrefixRouter {
	return &SlashPrefixRouter{state: StateIdle}
}

// Reset resets the internal state machine back to StateIdle.
func (r *SlashPrefixRouter) Reset() {
	r.state = StateIdle
	r.prefixBuf.Reset()
	r.commandBuf.Reset()
}

// ProcessByte processes a single byte through the prefix state machine.
func (r *SlashPrefixRouter) ProcessByte(b byte) RouterOutput {
	if b == 0x03 || b == 0x15 { // Ctrl+C or Ctrl+U
		r.Reset()
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
	}

	switch r.state {
	case StateIdle:
		if b == '/' {
			r.state = StateSlash
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		if b == '\r' || b == '\n' {
			return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
		}
		r.state = StatePassthrough
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}

	case StateSlash:
		if b == 'a' || b == 'A' {
			r.state = StateSlashA
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		if b == '/' {
			r.state = StateEscapeSlash
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		// Diverged from /ai (e.g. /help, /model, /review)
		r.state = StatePassthrough
		out := append(r.prefixBuf.Bytes(), b)
		r.prefixBuf.Reset()
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateSlashA:
		if b == 'i' || b == 'I' {
			r.state = StateSlashAI
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		// Diverged from /ai (e.g. /ask)
		r.state = StatePassthrough
		out := append(r.prefixBuf.Bytes(), b)
		r.prefixBuf.Reset()
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateSlashAI:
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			// Confirmed AI Control prefix
			r.state = StateControlCommand
			r.commandBuf.Reset()
			r.commandBuf.WriteString("/ai")
			r.commandBuf.WriteByte(b)
			r.prefixBuf.Reset()

			if b == '\r' || b == '\n' {
				cmd := strings.TrimSpace(r.commandBuf.String())
				r.Reset()
				return RouterOutput{Action: ActionControlCommand, ControlCmd: cmd}
			}
			return RouterOutput{Action: ActionNone}
		}
		// Diverged from /ai (e.g. /airplane)
		r.state = StatePassthrough
		out := append(r.prefixBuf.Bytes(), b)
		r.prefixBuf.Reset()
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateControlCommand:
		if b == '\r' || b == '\n' {
			cmd := strings.TrimSpace(r.commandBuf.String())
			r.Reset()
			return RouterOutput{Action: ActionControlCommand, ControlCmd: cmd}
		}
		if b == 0x7f || b == 0x08 { // Backspace
			if r.commandBuf.Len() > 0 {
				buf := r.commandBuf.Bytes()
				r.commandBuf.Reset()
				r.commandBuf.Write(buf[:len(buf)-1])
			}
			return RouterOutput{Action: ActionNone}
		}
		r.commandBuf.WriteByte(b)
		return RouterOutput{Action: ActionNone}

	case StateEscapeSlash:
		if b == 'a' || b == 'A' {
			r.state = StateEscapeSlashA
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		// Diverged from //ai (e.g. //comment)
		r.state = StatePassthrough
		out := append(r.prefixBuf.Bytes(), b)
		r.prefixBuf.Reset()
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateEscapeSlashA:
		if b == 'i' || b == 'I' {
			r.state = StateEscapeSlashAI
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		r.state = StatePassthrough
		out := append(r.prefixBuf.Bytes(), b)
		r.prefixBuf.Reset()
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateEscapeSlashAI:
		// Unescape "//ai" -> "/ai" + b to child process
		r.state = StatePassthrough
		out := append([]byte("/ai"), b)
		r.prefixBuf.Reset()
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StatePassthrough:
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
	}

	return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
}
