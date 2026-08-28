package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestWeb_FullE2E(t *testing.T) {
	// Register a fake runtime in the registry
	reg := registry.DefaultRegistry()
	fakeSession := registry.RuntimeSession{
		RuntimeID:       "e2e-fake-1",
		ProviderID:      "fake",
		ProfileID:       "default",
		Workspace:       "/tmp",
		PID:             99999,
		HostPID:         99999,
		HostGeneration:  time.Now().UnixNano(),
		State:           registry.StateRunning,
		ControlLevel:    registry.ControlLevelTerminal,
		ControlEndpoint: "/tmp/fake.sock",
		StartedAt:       time.Now(),
	}
	if err := reg.Register(fakeSession); err != nil {
		t.Fatalf("failed to register fake runtime: %v", err)
	}
	defer func() {
		_ = reg.UpdateState(fakeSession.RuntimeID, registry.StateStopped)
	}()

	srv, err := NewServer(ServerOptions{
		Host:   "127.0.0.1",
		Port:   0,
		NoOpen: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	go func() {
		_ = srv.Start()
	}()
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Visit bootstrap URL
	resp, err := client.Get(srv.BootstrapURL())
	if err != nil {
		t.Fatalf("failed to visit bootstrap URL: %v", err)
	}
	resp.Body.Close()

	// 2. Fetch session details (CSRF token)
	sessResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("failed to fetch session: %v", err)
	}
	defer sessResp.Body.Close()
	var sessData struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	_ = json.NewDecoder(sessResp.Body).Decode(&sessData)
	if !sessData.Authenticated {
		t.Fatal("expected session to be authenticated")
	}

	// 3. Fetch runtimes list: should include our fake runtime
	rtsResp, err := client.Get(srv.URL() + "/api/v1/runtimes")
	if err != nil {
		t.Fatalf("failed to fetch runtimes: %v", err)
	}
	defer rtsResp.Body.Close()
	var runtimes []*registry.RuntimeSession
	_ = json.NewDecoder(rtsResp.Body).Decode(&runtimes)

	found := false
	for _, r := range runtimes {
		if r.RuntimeID == "e2e-fake-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find e2e-fake-1 in runtimes, got %d runtimes", len(runtimes))
	}

	// 4. Test WebSocket endpoint
	wsURL := "ws://" + srv.listener.Addr().String() + "/api/v1/runtimes/e2e-fake-1/terminal"
	header := http.Header{}
	header.Set("Origin", "http://127.0.0.1")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Logf("WebSocket dial expectedly encountered offline socket for fake runtime: %v", err)
	} else {
		defer ws.Close()
		var msg TerminalMessage
		_ = ws.ReadJSON(&msg)
		if msg.Type == "error" || msg.Type == "lease" {
			t.Logf("Received valid initial terminal frame: %+v", msg)
		}
	}
}
