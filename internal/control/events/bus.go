package events

import (
	"sync"
)

// Bus provides thread-safe in-memory pub/sub routing and historical buffer for events.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
	history     map[string][]Event
	maxHistory  int
}

var (
	defaultBus *Bus
	busOnce    sync.Once
)

// DefaultBus returns the singleton event bus instance.
func DefaultBus() *Bus {
	busOnce.Do(func() {
		defaultBus = NewBus(200)
	})
	return defaultBus
}

// NewBus creates a new Event Bus with a max history size per runtime.
func NewBus(maxHistory int) *Bus {
	if maxHistory <= 0 {
		maxHistory = 200
	}
	return &Bus{
		subscribers: make(map[string][]chan Event),
		history:     make(map[string][]Event),
		maxHistory:  maxHistory,
	}
}

// Publish distributes an event to all interested subscribers and appends to history.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Append to history
	hist := b.history[e.RuntimeID]
	hist = append(hist, e)
	if len(hist) > b.maxHistory {
		hist = hist[len(hist)-b.maxHistory:]
	}
	b.history[e.RuntimeID] = hist

	// Broadcast to runtime subscribers and wildcard subscribers ("*")
	targets := append([]chan Event{}, b.subscribers[e.RuntimeID]...)
	targets = append(targets, b.subscribers["*"]...)

	for _, ch := range targets {
		select {
		case ch <- e:
		default:
			// Non-blocking drop if channel buffer is full
		}
	}
}

// Subscribe opens a stream of events for a runtime ID (or "*" for all runtimes).
// Returns the event channel and an unsubscribe function.
func (b *Bus) Subscribe(runtimeID string) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 64)
	b.subscribers[runtimeID] = append(b.subscribers[runtimeID], ch)

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subs := b.subscribers[runtimeID]
		for i, sub := range subs {
			if sub == ch {
				b.subscribers[runtimeID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsub
}

// GetHistory returns recent events for a given runtime ID.
func (b *Bus) GetHistory(runtimeID string, limit int) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hist, ok := b.history[runtimeID]
	if !ok {
		return nil
	}

	if limit <= 0 || limit > len(hist) {
		limit = len(hist)
	}

	out := make([]Event, limit)
	copy(out, hist[len(hist)-limit:])
	return out
}
