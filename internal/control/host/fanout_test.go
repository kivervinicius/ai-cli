package host

import (
	"net"
	"testing"
)

func TestBoundedFanoutRecordsSlowClientDrops(t *testing.T) {
	server, client := net.Pipe()
	fanout := NewBoundedFanout(1)
	fanout.AddClient(server)

	chunk := []byte("output")
	fanout.Broadcast(chunk)
	fanout.Broadcast(chunk)
	fanout.Broadcast(chunk)

	stats := fanout.Stats()
	if stats.DroppedChunks == 0 {
		t.Fatal("expected a slow client drop to be recorded")
	}
	if stats.DroppedBytes != stats.DroppedChunks*uint64(len(chunk)) {
		t.Fatalf("dropped bytes = %d, want %d bytes per dropped chunk", stats.DroppedBytes, len(chunk))
	}

	fanout.RemoveClient(server)
	_ = server.Close()
	_ = client.Close()
}
