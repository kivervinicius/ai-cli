package host

import (
	"sync"
)

// RingBuffer is a thread-safe circular byte buffer with a maximum capacity.
type RingBuffer struct {
	mu     sync.RWMutex
	buf    []byte
	size   int
	start  int
	length int
}

// NewRingBuffer creates a RingBuffer with the given maximum byte size.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 64 * 1024 // 64 KB default
	}
	return &RingBuffer{
		buf:  make([]byte, capacity),
		size: capacity,
	}
}

// Write appends data to the ring buffer.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n = len(p)
	if n >= r.size {
		// Truncate to keep the most recent bytes
		p = p[n-r.size:]
		copy(r.buf, p)
		r.start = 0
		r.length = r.size
		return n, nil
	}

	for _, b := range p {
		idx := (r.start + r.length) % r.size
		r.buf[idx] = b
		if r.length < r.size {
			r.length++
		} else {
			r.start = (r.start + 1) % r.size
		}
	}

	return n, nil
}

// Bytes returns a copy of the contiguous buffer content in chronological order.
func (r *RingBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]byte, r.length)
	for i := 0; i < r.length; i++ {
		out[i] = r.buf[(r.start+i)%r.size]
	}
	return out
}
