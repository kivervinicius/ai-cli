package websocketio

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWriterSerializesConcurrentWrites(t *testing.T) {
	var active int32
	var maxActive int32
	gate := NewWriter(func(v any) error {
		n := atomic.AddInt32(&active, 1)
		for {
			old := atomic.LoadInt32(&maxActive)
			if n <= old || atomic.CompareAndSwapInt32(&maxActive, old, n) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := gate.WriteJSON(i); err != nil {
				t.Errorf("WriteJSON: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&maxActive); got != 1 {
		t.Fatalf("expected serialized writes, max concurrent writes=%d", got)
	}
}
