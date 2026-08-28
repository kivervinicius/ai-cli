package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

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
	if err != nil || provResp.StatusCode != http.StatusOK {
		t.Errorf("providers endpoint failed: %v, status: %d", err, provResp.StatusCode)
	}
	provResp.Body.Close()

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
}
