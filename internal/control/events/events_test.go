package events

import (
	"testing"
	"time"
)

func TestEventBusPubSubAndHistory(t *testing.T) {
	bus := NewBus(10)

	runtimeID := "rt-test-events"
	ch, unsub := bus.Subscribe(runtimeID)
	defer unsub()

	evt1 := NewEvent(runtimeID, "codex", "work", EventAgentWorking, "Refactoring code", nil)
	bus.Publish(evt1)

	select {
	case received := <-ch:
		if received.Type != EventAgentWorking || received.Summary != "Refactoring code" {
			t.Errorf("unexpected event received: %+v", received)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for event on channel")
	}

	// Verify history
	hist := bus.GetHistory(runtimeID, 5)
	if len(hist) != 1 || hist[0].ID != evt1.ID {
		t.Errorf("unexpected history: %+v", hist)
	}
}
