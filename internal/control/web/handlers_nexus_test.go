package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func newTestClient(t *testing.T) (*http.Client, *Server) {
	t.Helper()
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	go func() { _ = srv.Start() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	time.Sleep(50 * time.Millisecond)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	resp, err := client.Get(srv.BootstrapURL())
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	_ = resp.Body.Close()
	return client, srv
}

// csrfClient returns an authenticated client plus the CSRF token.
func csrfClient(t *testing.T) (*http.Client, *Server, string) {
	t.Helper()
	client, srv := newTestClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	var sess struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&sess)
	resp.Body.Close()
	if !sess.Authenticated || sess.CSRFToken == "" {
		t.Fatal("session not authenticated / no CSRF token")
	}
	return client, srv, sess.CSRFToken
}

func TestNexusProjectsAndAgentsAPI(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	// Create a project from a temp dir.
	projDir := t.TempDir()
	body, _ := json.Marshal(map[string]string{"name": "Omega API", "path": projDir})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	var proj store.Project
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&proj)
	resp.Body.Close()
	if proj.ID == "" || len(proj.ID) < 5 {
		t.Fatalf("invalid project id: %q", proj.ID)
	}
	if proj.MaestroMode != store.MaestroAssist {
		t.Errorf("default maestro mode expected ASSIST, got %q", proj.MaestroMode)
	}

	// List projects.
	resp, err = client.Get(base + "/api/v1/projects")
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	var projects []store.Project
	_ = json.NewDecoder(resp.Body).Decode(&projects)
	resp.Body.Close()
	if len(projects) < 1 {
		t.Fatal("expected at least one project")
	}

	// Create an agent in the project.
	abody, _ := json.Marshal(map[string]string{"name": "Backend Developer"})
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+proj.ID+"/agents", bytes.NewReader(abody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	var agent store.Agent
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent expected 201, got %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&agent)
	resp.Body.Close()
	if agent.ID == "" || len(agent.ID) < 5 {
		t.Fatalf("invalid agent id: %q", agent.ID)
	}

	// List agents.
	resp, err = client.Get(base + "/api/v1/projects/" + proj.ID + "/agents")
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	var agents []store.Agent
	_ = json.NewDecoder(resp.Body).Decode(&agents)
	resp.Body.Close()
	if len(agents) != 1 || agents[0].ID != agent.ID {
		t.Fatalf("expected agent %s, got %+v", agent.ID, agents)
	}

	// Agent detail (generations/lineage empty).
	resp, err = client.Get(base + "/api/v1/agents/" + agent.ID)
	if err != nil {
		t.Fatalf("agent detail: %v", err)
	}
	var detail map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	resp.Body.Close()
	if detail["agent"] == nil || detail["generations"] == nil {
		t.Fatalf("agent detail missing fields: %+v", detail)
	}

	// Layout round-trip.
	lbody, _ := json.Marshal(map[string]string{"layout": `{"openAgents":["` + agent.ID + `"]}`})
	req, _ = http.NewRequest(http.MethodPut, base+"/api/v1/projects/"+proj.ID+"/layout", bytes.NewReader(lbody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("save layout: %v", err)
	}
	resp.Body.Close()
	resp, err = client.Get(base + "/api/v1/projects/" + proj.ID + "/layout")
	if err != nil {
		t.Fatalf("get layout: %v", err)
	}
	var layoutResp map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&layoutResp)
	resp.Body.Close()
	if layoutResp["layout"] == "" || !bytes.Contains([]byte(layoutResp["layout"]), []byte(agent.ID)) {
		t.Fatalf("layout roundtrip failed: %q", layoutResp["layout"])
	}

	// Agent terminal WS without active runtime must fail cleanly (404).
	wsResp, err := client.Get(base + "/api/v1/agents/" + agent.ID + "/terminal")
	if err != nil {
		t.Fatalf("agent terminal (no runtime): %v", err)
	}
	_ = wsResp.Body.Close()
	if wsResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for terminal without runtime, got %d", wsResp.StatusCode)
	}

	// Delete agent.
	dreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/agents/"+agent.ID, nil)
	dreq.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(dreq)
	if err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("delete agent expected 200, got %d", resp.StatusCode)
	}

	// Delete project.
	pdreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/projects/"+proj.ID, nil)
	pdreq.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(pdreq)
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}
	resp.Body.Close()
}
