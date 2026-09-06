package host

import (
	"bytes"
	"strings"
)

type PrefixState int

const (
	StateIdle PrefixState = iota
	StateBuffering
	StateControlCommand
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

// SlashPrefixRouter handles input bytes character by character, buffering ambiguous slash prefixes
// (/nexus, /ai, //nexus, //ai) and instantly forwarding non-matching bytes to the child process.
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

func isPrefixOfKnown(s string) bool {
	targets := []string{"/nexus", "/ai", "//nexus", "//ai"}
	for _, t := range targets {
		if strings.HasPrefix(t, s) {
			return true
		}
	}
	return false
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
			r.state = StateBuffering
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		if b == '\r' || b == '\n' {
			return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
		}
		r.state = StatePassthrough
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}

	case StateBuffering:
		// Check if delimiter encountered
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			trimmedPrefix := strings.ToLower(strings.TrimSpace(r.prefixBuf.String()))
			if trimmedPrefix == "/nexus" || trimmedPrefix == "/ai" {
				// Confirmed command prefix
				r.state = StateControlCommand
				r.commandBuf.Reset()
				r.commandBuf.WriteString(r.prefixBuf.String())
				r.commandBuf.WriteByte(b)
				r.prefixBuf.Reset()

				if b == '\r' || b == '\n' {
					cmd := strings.TrimSpace(r.commandBuf.String())
					r.Reset()
					return RouterOutput{Action: ActionControlCommand, ControlCmd: cmd}
				}
				return RouterOutput{Action: ActionNone}
			}

			if trimmedPrefix == "//nexus" {
				r.state = StatePassthrough
				out := append([]byte("/nexus"), b)
				r.prefixBuf.Reset()
				if b == '\r' || b == '\n' {
					r.state = StateIdle
				}
				return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}
			}

			if trimmedPrefix == "//ai" {
				r.state = StatePassthrough
				out := append([]byte("/ai"), b)
				r.prefixBuf.Reset()
				if b == '\r' || b == '\n' {
					r.state = StateIdle
				}
				return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}
			}

			// Diverged delimiter
			r.state = StatePassthrough
			out := append(r.prefixBuf.Bytes(), b)
			r.prefixBuf.Reset()
			if b == '\r' || b == '\n' {
				r.state = StateIdle
			}
			return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}
		}

		r.prefixBuf.WriteByte(b)
		candidate := strings.ToLower(r.prefixBuf.String())
		if isPrefixOfKnown(candidate) {
			return RouterOutput{Action: ActionNone}
		}

		// Diverged from all known slash prefixes
		r.state = StatePassthrough
		out := make([]byte, r.prefixBuf.Len())
		copy(out, r.prefixBuf.Bytes())
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

	case StatePassthrough:
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
	}

	return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
}
