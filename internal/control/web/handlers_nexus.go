package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// NexusHandler serves the Nexus product API (projects, agents, generations,
// lineage, layouts) on top of the shared control core.
type NexusHandler struct {
	auth  *AuthManager
	nexus *nexus.Nexus
}

func NewNexusHandler(auth *AuthManager) *NexusHandler {
	return &NexusHandler{auth: auth, nexus: nexus.Default()}
}

// handleProjectsList GET|POST /api/v1/projects
func (h *NexusHandler) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := st.ListProjects()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
			writeError(w, http.StatusBadRequest, "path is required")
			return
		}
		canonical, err := store.CanonicalPath(body.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		proj, err := st.CreateProject(store.Project{Name: body.Name, CanonicalPath: canonical})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, proj)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectDetail GET/PATCH/DELETE /api/v1/projects/{id}
func (h *NexusHandler) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		proj, err := st.GetProject(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		layout, _ := st.GetLayout(id)
		writeJSON(w, http.StatusOK, map[string]any{"project": proj, "layout": layout})

	case http.MethodPatch:
		proj, err := st.GetProject(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		var body struct {
			Name             *string `json:"name"`
			MaestroMode      *string `json:"maestro_mode"`
			DefaultIsolation *string `json:"default_isolation"`
			DefaultBranch    *string `json:"default_branch"`
			ResourcePolicy   *string `json:"resource_policy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			proj.Name = *body.Name
			proj.Slug = store.Slugify(*body.Name)
		}
		if body.MaestroMode != nil {
			proj.MaestroMode = *body.MaestroMode
		}
		if body.DefaultIsolation != nil {
			proj.DefaultIsolation = *body.DefaultIsolation
		}
		if body.DefaultBranch != nil {
			proj.DefaultBranch = *body.DefaultBranch
		}
		if body.ResourcePolicy != nil {
			proj.ResourcePolicy = *body.ResourcePolicy
		}
		if err := st.UpdateProject(proj); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, proj)

	case http.MethodDelete:
		if err := st.DeleteProject(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectLayout GET/PUT /api/v1/projects/{id}/layout
func (h *NexusHandler) handleProjectLayout(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodGet:
		layout, err := st.GetLayout(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"layout": layout})
	case http.MethodPut:
		var body struct {
			Layout string `json:"layout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if err := st.SaveLayout(id, body.Layout); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentsList GET /api/v1/projects/{projectID}/agents
func (h *NexusHandler) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	agents, err := st.ListAgents(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

// handleAgentCreate POST /api/v1/projects/{projectID}/agents  {name}
func (h *NexusHandler) handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: projectID, Name: body.Name, Role: body.Role})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

// handleAgentDetail GET/PATCH/DELETE /api/v1/agents/{id}
func (h *NexusHandler) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		generations, _ := st.ListGenerations(id)
		lineage, _ := st.ListLineage(id)
		revisions, _ := st.ListRevisions(id)
		effectiveState, _ := h.nexus.EffectiveAgentState(id)
		writeJSON(w, http.StatusOK, map[string]any{
			"agent":           agent,
			"generations":     generations,
			"lineage":         lineage,
			"revisions":       revisions,
			"effective_state": effectiveState,
			"recoverable":     effectiveState == store.AgentRecoverable,
		})
	case http.MethodPatch:
		var body struct {
			Name *string `json:"name"`
			Role *string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			agent.Name = *body.Name
		}
		if body.Role != nil {
			agent.Role = *body.Role
		}
		if err := st.UpdateAgent(agent); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, agent)
	case http.MethodDelete:
		if err := st.DeleteAgent(id, agent.ProjectID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentStart POST /api/v1/agents/{id}/start {provider?, profile?}
func (h *NexusHandler) handleAgentStart(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var body struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	sess, err := h.nexus.StartAgent(context.Background(), id, body.Provider, body.Profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runtime": sess})
}

// handleAgentStop POST /api/v1/agents/{id}/stop
func (h *NexusHandler) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	if err := h.nexus.StopAgent(context.Background(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// handleAgentRecover POST /api/v1/agents/{id}/recover
func (h *NexusHandler) handleAgentRecover(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	sess, err := h.nexus.RecoverAgent(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runtime": sess})
}

// resolveAgentRuntimeID maps an agent to its current runtime generation ID.
func (h *NexusHandler) resolveAgentRuntimeID(agentID string) (string, error) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return "", err
	}
	gen, err := st.CurrentGeneration(agentID)
	if err != nil {
		return "", err
	}
	return gen.RuntimeID, nil
}

func projectIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/projects/"), "/")
	if len(parts) >= 1 && strings.HasSuffix(path, "/agents") {
		return parts[0]
	}
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func agentIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/agents/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
