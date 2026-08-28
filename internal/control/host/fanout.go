package host

import (
	"net"
	"sync"
)

type clientWorker struct {
	conn     net.Conn
	queue    chan []byte
	doneChan chan struct{}
}

// BoundedFanout manages isolated per-client delivery queues so slow observers
// never block the supervised child process terminal stream.
type BoundedFanout struct {
	mu      sync.RWMutex
	clients map[net.Conn]*clientWorker
	bufCap  int
}

func NewBoundedFanout(bufCap int) *BoundedFanout {
	if bufCap <= 0 {
		bufCap = 256
	}
	return &BoundedFanout{
		clients: make(map[net.Conn]*clientWorker),
		bufCap:  bufCap,
	}
}

func (f *BoundedFanout) AddClient(conn net.Conn) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.clients[conn]; exists {
		return
	}

	worker := &clientWorker{
		conn:     conn,
		queue:    make(chan []byte, f.bufCap),
		doneChan: make(chan struct{}),
	}
	f.clients[conn] = worker

	go f.pumpClient(worker)
}

func (f *BoundedFanout) RemoveClient(conn net.Conn) {
	f.mu.Lock()
	worker, exists := f.clients[conn]
	if exists {
		delete(f.clients, conn)
	}
	f.mu.Unlock()

	if exists && worker != nil {
		select {
		case <-worker.doneChan:
		default:
			close(worker.doneChan)
		}
	}
}

func (f *BoundedFanout) Broadcast(data []byte) {
	if len(data) == 0 {
		return
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Clone chunk so goroutines don't race on underlying buffer
	chunk := make([]byte, len(data))
	copy(chunk, data)

	for _, w := range f.clients {
		select {
		case w.queue <- chunk:
		default:
			// Slow consumer queue is full: drop chunk rather than blocking child process
		}
	}
}

func (f *BoundedFanout) pumpClient(w *clientWorker) {
	for {
		select {
		case <-w.doneChan:
			return
		case data, ok := <-w.queue:
			if !ok {
				return
			}
			if _, err := w.conn.Write(data); err != nil {
				return
			}
		}
	}
}

func (f *BoundedFanout) Close() {
	f.mu.Lock()
	workers := make([]*clientWorker, 0, len(f.clients))
	for _, w := range f.clients {
		workers = append(workers, w)
	}
	f.clients = make(map[net.Conn]*clientWorker)
	f.mu.Unlock()

	for _, w := range workers {
		select {
		case <-w.doneChan:
		default:
			close(w.doneChan)
		}
	}
}
