package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
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
	n := nexus.Default()
	// Wire runtime change notifications to the terminal broker (Gate 4).
	broker := DefaultBroker()
	n.SetRuntimeObservers(
		broker.NotifyRuntimeChanged,
		broker.NotifyAgentState,
		broker.NotifyContinuity,
	)
	return &NexusHandler{auth: auth, nexus: n}
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
		// Touch MRU on access (P1 Project MRU).
		_ = st.TouchProject(id)
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
		if err := h.nexus.DeleteProject(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
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
		if err := h.nexus.DeleteAgent(id, agent.ProjectID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentStart POST /api/v1/agents/{id}/start. Provider selection is
// intentionally read only here: it must be persisted through Resources first.
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

	provider, profile, err := h.nexus.ResolveStartParams(id, body.Provider, body.Profile)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	sess, err := h.nexus.StartAgent(context.Background(), id, provider, profile)
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

// handleAgentConfigGet GET /api/v1/agents/{id}/config
func (h *NexusHandler) handleAgentConfigGet(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var cfg nexus.AgentConfig
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			cfg, _ = nexus.ParseAgentConfig(rev.Config)
		}
	}
	revs, _ := st.ListRevisions(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"config":    cfg,
		"revision":  agent.CurrentRevisionID,
		"revisions": revs,
	})
}

// handleAgentConfigApply POST /api/v1/agents/{id}/config/apply
func (h *NexusHandler) handleAgentConfigApply(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var cfg nexus.AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config body")
		return
	}
	impact, err := h.nexus.SafeApply(context.Background(), id, cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}

// handleAgentConfigImpact POST /api/v1/agents/{id}/config/impact
func (h *NexusHandler) handleAgentConfigImpact(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var cfg nexus.AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config body")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var current nexus.AgentConfig
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			current, _ = nexus.ParseAgentConfig(rev.Config)
		}
	}
	impact := nexus.AnalyzeImpact(current, cfg)
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}

// handleResourcesList GET /api/v1/resources — returns available provider accounts
// and the scheduler recommendation for the current context.
func (h *NexusHandler) handleResourcesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	accounts, err := h.nexus.ListResources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []nexus.ProviderAccount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"policy":   "BALANCED",
	})
}

// handleResourceSelect POST /api/v1/resources/select — run the scheduler
// and return an explainable decision.
func (h *NexusHandler) handleResourceSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
		Policy   string `json:"policy"`
		AgentID  string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(body.AgentID) == "" || strings.TrimSpace(body.Provider) == "" || strings.TrimSpace(body.Profile) == "" {
		writeError(w, http.StatusBadRequest, "agent_id, provider and profile are required")
		return
	}
	allocation, err := h.nexus.AllocateResource(context.Background(), body.AgentID, body.Provider, body.Profile, nexus.SchedulerPolicy(body.Policy))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, allocation)
}

// handleMaestroStatus GET /api/v1/maestro — returns Maestro integration status.
func (h *NexusHandler) handleMaestroStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	status := client.Status()
	writeJSON(w, http.StatusOK, status)
}

// handleMaestroAdvice POST /api/v1/maestro/advice — request Maestro recommendations.
func (h *NexusHandler) handleMaestroAdvice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		ProjectID string `json:"project_id"`
		AgentID   string `json:"agent_id"`
		Intent    string `json:"intent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	client := nexus.NewMaestroClient()
	ctx := nexus.AdviceContext{
		ProjectID: body.ProjectID,
		AgentID:   body.AgentID,
	}
	resp, err := client.GetAdvice(ctx, body.Intent)
	if err != nil {
		// Return degraded response, not 500.
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":     "OFF",
			"error":    err.Error(),
			"degraded": true,
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSystemUpdates GET /api/v1/system/updates — returns status of Nexus & Maestro versions.
func (h *NexusHandler) handleSystemUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	mStatus := client.Status()
	maestroVer := "unknown"
	if mStatus.Capabilities != nil {
		maestroVer = mStatus.Capabilities.Version
	}

	// Check npm registry for latest maestro version if available
	latestMaestroVer := maestroVer
	updateAvailable := false
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "view", "@iapro/orquestrador-maestro-cli", "version")
		if out, err := cmd.Output(); err == nil {
			latestMaestroVer = strings.TrimSpace(string(out))
			if latestMaestroVer != "" && maestroVer != "unknown" && latestMaestroVer != maestroVer {
				updateAvailable = true
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nexus_version":          buildinfo.Version,
		"nexus_commit":           buildinfo.Commit,
		"nexus_build_date":       buildinfo.BuildDate,
		"maestro_version":        maestroVer,
		"maestro_latest_version": latestMaestroVer,
		"maestro_available":      mStatus.Available,
		"update_available":       updateAvailable,
	})
}

// handleSystemUpdate POST /api/v1/system/update — triggers update routine for Nexus & Maestro.
func (h *NexusHandler) handleSystemUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeError(w, http.StatusNotImplemented, "system update not yet implemented")
}

// handleMissionsList GET /api/v1/projects/{id}/missions — list missions for a project.
func (h *NexusHandler) handleMissionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
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
	missions, err := st.ListMissions(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"missions": missions})
}

// handleMissionCreate POST /api/v1/projects/{id}/missions — create a mission.
func (h *NexusHandler) handleMissionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
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
		Name        string `json:"name"`
		Description string `json:"description"`
		Goal        string `json:"goal"`
		Scope       string `json:"scope"`
		RiskLevel   string `json:"risk_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	m := &store.Mission{
		ProjectID:   projectID,
		Name:        body.Name,
		Description: body.Description,
		Goal:        body.Goal,
		Scope:       body.Scope,
		RiskLevel:   body.RiskLevel,
	}
	if err := st.CreateMission(m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// handleMissionDetail GET/PATCH/DELETE /api/v1/missions/{id}
func (h *NexusHandler) handleMissionDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, err := st.GetMission(missionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		tasks, _ := st.ListTasks(missionID)
		assignments, _ := st.ListAssignments(missionID)
		total, pending, active, completed, failed, _ := st.MissionStats(missionID)
		writeJSON(w, http.StatusOK, map[string]any{
			"mission":     m,
			"tasks":       tasks,
			"assignments": assignments,
			"stats":       map[string]int{"total": total, "pending": pending, "active": active, "completed": completed, "failed": failed},
		})

	case http.MethodPatch:
		m, err := st.GetMission(missionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			Goal        *string `json:"goal"`
			Scope       *string `json:"scope"`
			RiskLevel   *string `json:"risk_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			m.Name = *body.Name
		}
		if body.Description != nil {
			m.Description = *body.Description
		}
		if body.Status != nil {
			m.Status = *body.Status
		}
		if body.Goal != nil {
			m.Goal = *body.Goal
		}
		if body.Scope != nil {
			m.Scope = *body.Scope
		}
		if body.RiskLevel != nil {
			m.RiskLevel = *body.RiskLevel
		}
		if err := st.UpdateMission(m); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, m)

	case http.MethodDelete:
		if err := st.DeleteMission(missionID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMissionTaskCreate POST /api/v1/missions/{id}/tasks — add a task to a mission.
func (h *NexusHandler) handleMissionTaskCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Kind         string `json:"kind"`
		Priority     int    `json:"priority"`
		Dependencies string `json:"dependencies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := &store.MissionTask{
		MissionID:    missionID,
		Name:         body.Name,
		Description:  body.Description,
		Kind:         body.Kind,
		Priority:     body.Priority,
		Dependencies: body.Dependencies,
	}
	if err := st.CreateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// handleMissionAssign POST /api/v1/missions/{id}/assign — assign an agent to a task.
func (h *NexusHandler) handleMissionAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		TaskID  string `json:"task_id"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	a := &store.MissionAssignment{
		MissionID: missionID,
		TaskID:    body.TaskID,
		AgentID:   body.AgentID,
	}
	if err := st.CreateAssignment(a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
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

func missionIDFromPath(path string) string {
	// Handle /api/v1/missions/{id} and /api/v1/missions/{id}/tasks and /api/v1/missions/{id}/assign
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/missions/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
