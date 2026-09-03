package notify

import "testing"

func TestRecorderCapturesPayload(t *testing.T) {
	rec := &Recorder{}
	SetDefault(rec)
	t.Cleanup(func() { SetDefault(nil) })

	if err := Default().Notify(Payload{Title: "Nexus · demo", Body: "Deseja continuar?", Tag: "fp1"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(rec.Payloads))
	}
	if rec.Payloads[0].Body != "Deseja continuar?" {
		t.Fatalf("unexpected body: %q", rec.Payloads[0].Body)
	}
}

func TestThrottleSuppressesSameTag(t *testing.T) {
	rec := &Recorder{}
	n := NewThrottled(rec, 0) // uses default 30s
	_ = n.Notify(Payload{Title: "a", Body: "b", Tag: "same"})
	_ = n.Notify(Payload{Title: "a", Body: "b", Tag: "same"})
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected throttle to keep 1 notify, got %d", len(rec.Payloads))
	}
}
