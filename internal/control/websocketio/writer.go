// Package websocketio provides the single-writer serialization required by
// Gorilla WebSocket connections. Gorilla permits one concurrent reader and one
// concurrent writer, but not multiple writers.
package websocketio

import "sync"

// Writer serializes calls to a connection-specific JSON write function.
type Writer struct {
	mu      sync.Mutex
	writeFn func(any) error
}

// NewWriter creates a serialized JSON writer around writeFn.
func NewWriter(writeFn func(any) error) *Writer {
	return &Writer{writeFn: writeFn}
}

// WriteJSON guarantees that at most one goroutine calls the underlying writer
// at a time.
func (w *Writer) WriteJSON(v any) error {
	if w == nil || w.writeFn == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeFn(v)
}
