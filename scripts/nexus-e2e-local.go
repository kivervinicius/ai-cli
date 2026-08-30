// Command nexus-e2e-local performs a real local Direct Work smoke test against
// a running Nexus instance and an already-authenticated provider/profile.
//
// Usage:
//
//	NEXUS_BOOTSTRAP_URL='http://127.0.0.1:PORT/?token=...' \
//	NEXUS_E2E_PROJECT_PATH=/path/to/git/project \
//	NEXUS_E2E_PROVIDER=claude \
//	NEXUS_E2E_PROFILE=default \
//	go run ./scripts/nexus-e2e-local.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type apiClient struct {
	base string
	csrf string
	http *http.Client
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "NEXUS_E2E_FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("NEXUS_E2E_PASS: Direct Work reached a real provider runtime and produced terminal output")
}

func run(ctx context.Context) error {
	bootstrap := strings.TrimSpace(os.Getenv("NEXUS_BOOTSTRAP_URL"))
	if bootstrap == "" {
		return fmt.Errorf("NEXUS_BOOTSTRAP_URL is required; use the one-time URL printed by the running Nexus")
	}
	u, err := url.Parse(bootstrap)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid NEXUS_BOOTSTRAP_URL")
	}
	base := u.Scheme + "://" + u.Host
	projectPath := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROJECT_PATH"))
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("project path: %w", err)
	}

	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, bootstrap, nil)
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("bootstrap returned HTTP %d", resp.StatusCode)
	}

	api := &apiClient{base: base, http: hc}
	var session struct {
		Authenticated bool   `json:"authenticated"`
		CSRF          string `json:"csrf_token"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/session", nil, &session); err != nil {
		return err
	}
	if !session.Authenticated || session.CSRF == "" {
		return fmt.Errorf("bootstrap did not establish an authenticated session")
	}
	api.csrf = session.CSRF

	stamp := time.Now().UTC().Format("20060102-150405")
	var project struct {
		ID string `json:"id"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "Nexus E2E " + stamp,
		"path": projectPath,
	}, &project); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	if project.ID == "" {
		return fmt.Errorf("create project returned empty id")
	}

	var resources struct {
		Accounts []struct {
			ID            string `json:"id"`
			Provider      string `json:"provider"`
			Profile       string `json:"profile"`
			Authenticated bool   `json:"authenticated"`
			Available     bool   `json:"available"`
			RateLimited   bool   `json:"rate_limited"`
		} `json:"accounts"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/resources", nil, &resources); err != nil {
		return err
	}
	wantProvider := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROVIDER"))
	wantProfile := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROFILE"))
	var provider, profile string
	for _, account := range resources.Accounts {
		if !account.Authenticated || !account.Available || account.RateLimited {
			continue
		}
		if wantProvider != "" && account.Provider != wantProvider {
			continue
		}
		if wantProfile != "" && account.Profile != wantProfile {
			continue
		}
		provider, profile = account.Provider, account.Profile
		break
	}
	if provider == "" {
		return fmt.Errorf("no eligible authenticated provider/profile matches provider=%q profile=%q", wantProvider, wantProfile)
	}

	var agent struct {
		ID string `json:"id"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/projects/"+url.PathEscape(project.ID)+"/agents", map[string]any{
		"name": "E2E Direct " + provider,
		"role": "developer",
	}, &agent); err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	if agent.ID == "" {
		return fmt.Errorf("create agent returned empty id")
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = api.do(cleanupCtx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agent.ID)+"/stop", map[string]any{}, nil)
		if os.Getenv("NEXUS_E2E_KEEP") != "1" {
			_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/agents/"+url.PathEscape(agent.ID), nil, nil)
			_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/projects/"+url.PathEscape(project.ID), nil, nil)
		}
	}()

	if err := api.do(ctx, http.MethodPost, "/api/v1/resources/select", map[string]any{
		"provider": provider,
		"profile":  profile,
		"policy":   "MANUAL",
		"agent_id": agent.ID,
	}, nil); err != nil {
		return fmt.Errorf("persist resource: %w", err)
	}

	var started struct {
		Runtime struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
			State     string `json:"state"`
		} `json:"runtime"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agent.ID)+"/start", map[string]any{}, &started); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}
	if started.Runtime.RuntimeID == "" || started.Runtime.AgentID != agent.ID {
		return fmt.Errorf("runtime identity mismatch runtime=%q runtime.agent_id=%q agent=%q", started.Runtime.RuntimeID, started.Runtime.AgentID, agent.ID)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := wsScheme + "://" + u.Host + "/api/v1/agents/" + url.PathEscape(agent.ID) + "/terminal"
	header := http.Header{}
	header.Set("Origin", base)
	baseURL, _ := url.Parse(base)
	if cookies := jar.Cookies(baseURL); len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	ws, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("agent terminal websocket: %w", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(45 * time.Second))
	control := false
	for !control {
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for terminal CONTROL lease: %w", err)
		}
		if msg["type"] == "lease" && msg["role"] == "CONTROL" {
			control = true
		}
	}

	prompt := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROMPT"))
	if prompt == "" {
		prompt = "Reply with exactly NEXUS_E2E_OK and nothing else."
	}
	if err := ws.WriteJSON(map[string]any{"type": "input", "data": prompt + "\n"}); err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}

	var providerOutput strings.Builder
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		_ = ws.SetReadDeadline(deadline)
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for provider output after prompt: %w", err)
		}
		if msg["type"] == "output" {
			if s, ok := msg["data"].(string); ok && strings.TrimSpace(s) != "" {
				providerOutput.WriteString(s)
				if strings.TrimSpace(providerOutput.String()) != "" {
					break
				}
			}
		}
	}
	if strings.TrimSpace(providerOutput.String()) == "" {
		return fmt.Errorf("provider produced no terminal output after prompt")
	}

	var detail struct {
		Generations []struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
		} `json:"generations"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(agent.ID), nil, &detail); err != nil {
		return err
	}
	found := false
	for _, generation := range detail.Generations {
		if generation.RuntimeID == started.Runtime.RuntimeID && generation.AgentID == agent.ID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("runtime generation was not durably linked back to the Agent")
	}
	return nil
}

func (a *apiClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead && a.csrf != "" {
		req.Header.Set("X-CSRF-Token", a.csrf)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return nil
}
