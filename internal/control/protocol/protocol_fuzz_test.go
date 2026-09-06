package protocol

import (
	"encoding/json"
	"testing"
)

func FuzzProtocolRequestResponse(f *testing.F) {
	seeds := []string{
		`{"version":1,"command":"ping","timestamp":"2026-08-28T00:00:00Z"}`,
		`{"version":1,"command":"input","payload":{"data":"abc"},"timestamp":"2026-08-28T00:00:00Z"}`,
		`{"version":1,"command":"resize","payload":{"rows":24,"cols":80},"timestamp":"2026-08-28T00:00:00Z"}`,
		`{"version":1,"command":"unknown_command","payload":{},"timestamp":"2026-08-28T00:00:00Z"}`,
		`{"version":99999,"command":"ping","timestamp":"2026-08-28T00:00:00Z"}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var req Request
		_ = json.Unmarshal([]byte(data), &req)
		var resp Response
		_ = json.Unmarshal([]byte(data), &resp)
		// Re-marshal must never panic.
		_, _ = json.Marshal(req)
		_, _ = json.Marshal(resp)
	})
}
