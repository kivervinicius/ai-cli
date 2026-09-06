package web

import "testing"

func TestIsProtocolControlFrame(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "lease request", in: `{"version":1,"id":"req-1","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}`, want: true},
		{name: "ok response", in: `{"version":1,"ok":true,"data":"\"lease_acquired\""}`, want: true},
		{name: "error response", in: `{"version":1,"ok":false,"error":"boom"}`, want: true},
		{name: "pty text", in: "hello from codex\n", want: false},
		{name: "json-looking prompt", in: `{"note":"user typed this"}`, want: false},
		{name: "empty", in: "   ", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isProtocolControlFrame([]byte(tc.in)); got != tc.want {
				t.Fatalf("isProtocolControlFrame(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestScrubProtocolFramesFromOutput(t *testing.T) {
	in := "welcome\n" +
		`{"version":1,"id":"req-1","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}` + "\n" +
		"real pty line\n" +
		`{"version":1,"ok":true,"data":"\"lease_acquired\""}` + "\n" +
		`{"note":"user typed this"}` + "\n"
	got := scrubProtocolFramesFromOutput(in)
	want := "welcome\nreal pty line\n{\"note\":\"user typed this\"}\n"
	if got != want {
		t.Fatalf("scrubProtocolFramesFromOutput()\n got %q\nwant %q", got, want)
	}
	if scrubProtocolFramesFromOutput(`{"version":1,"command":"lease_acquire"}`) != "" {
		t.Fatal("expected empty when history is only protocol frames")
	}
}
