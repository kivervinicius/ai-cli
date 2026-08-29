package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestMain(m *testing.M) {
	testDir, err := os.MkdirTemp("", "ai-control-web-test-*")
	if err == nil {
		defer os.RemoveAll(testDir)
		_ = os.Setenv("AI_MANAGER_DATA_DIR", testDir)
		_ = os.Setenv("AI_CLI_DATA_DIR", testDir)
	}
	registry.ResetDefaultRegistryForTest()
	code := m.Run()
	registry.ResetDefaultRegistryForTest()
	os.Exit(code)
}

func TestServer_BootstrapAndAuth(t *testing.T) {
	srv, err := NewServer(ServerOptions{
		Host: "127.0.0.1",
		Port: 0, // Random port
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

	// 1. Visit bootstrap URL with token: should exchange and set cookie
	bootstrapURL := srv.BootstrapURL()
	resp, err := client.Get(bootstrapURL)
	if err != nil {
		t.Fatalf("failed to visit bootstrap URL: %v", err)
	}
	defer resp.Body.Close()

	// 2. Check session endpoint
	sessResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("failed to query session endpoint: %v", err)
	}
	defer sessResp.Body.Close()

	var sessData struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&sessData); err != nil {
		t.Fatalf("failed to decode session response: %v", err)
	}

	if !sessData.Authenticated || sessData.CSRFToken == "" {
		t.Errorf("expected session to be authenticated with CSRF token, got %+v", sessData)
	}

	// 3. Query health endpoint
	healthResp, err := client.Get(srv.URL() + "/api/v1/health")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Errorf("health endpoint failed: %v, status: %d", err, healthResp.StatusCode)
	}
	healthResp.Body.Close()

	// 4. Query workspaces endpoint
	wsResp, err := client.Get(srv.URL() + "/api/v1/workspaces")
	if err != nil || wsResp.StatusCode != http.StatusOK {
		t.Errorf("workspaces endpoint failed: %v, status: %d", err, wsResp.StatusCode)
	}
	wsResp.Body.Close()

	// 5. Query providers endpoint
	provResp, err := client.Get(srv.URL() + "/api/v1/providers")
	if err != nil {
		t.Errorf("providers endpoint request failed: %v", err)
	} else {
		defer provResp.Body.Close()
		if provResp.StatusCode != http.StatusOK {
			t.Errorf("providers endpoint failed: status: %d", provResp.StatusCode)
		}
	}

	// 6. Test Origin enforcement
	badReq, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	badReq.Header.Set("Origin", "http://malicious-site.com")
	badResp, err := client.Do(badReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for untrusted Origin, got %d", badResp.StatusCode)
	}

	// 7. Test prefix spoofed Origin enforcement
	evilReq, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	evilReq.Header.Set("Origin", "http://localhost.evil.com")
	evilResp, err := client.Do(evilReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer evilResp.Body.Close()
	if evilResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for prefix spoofed Origin http://localhost.evil.com, got %d", evilResp.StatusCode)
	}

	evilReq2, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	evilReq2.Header.Set("Origin", "http://127.0.0.1.attacker.com")
	evilResp2, err := client.Do(evilReq2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer evilResp2.Body.Close()
	if evilResp2.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for prefix spoofed Origin http://127.0.0.1.attacker.com, got %d", evilResp2.StatusCode)
	}

	// 8. Test unauthenticated client: GET on API routes MUST return 401 Unauthorized
	unauthClient := &http.Client{}
	unauthEndpoints := []string{
		"/api/v1/workspaces",
		"/api/v1/runtimes",
		"/api/v1/providers",
		"/api/v1/profiles",
		"/api/v1/events",
	}
	for _, ep := range unauthEndpoints {
		resp, err := unauthClient.Get(srv.URL() + ep)
		if err != nil {
			t.Fatalf("GET %s failed: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated GET %s, got %d", ep, resp.StatusCode)
		}
	}

	// 9. Test public endpoints without authentication: health and session must succeed (200)
	pubHealthResp, err := unauthClient.Get(srv.URL() + "/api/v1/health")
	if err != nil || pubHealthResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for public health endpoint, got %d", pubHealthResp.StatusCode)
	}
	pubHealthResp.Body.Close()

	pubSessResp, err := unauthClient.Get(srv.URL() + "/api/v1/session")
	if err != nil || pubSessResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for public session endpoint, got %d", pubSessResp.StatusCode)
	}
	var unauthSessData struct {
		Authenticated bool `json:"authenticated"`
	}
	_ = json.NewDecoder(pubSessResp.Body).Decode(&unauthSessData)
	pubSessResp.Body.Close()
	if unauthSessData.Authenticated {
		t.Errorf("expected unauthenticated session for unauth client, got true")
	}
}
