package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/handoff"
	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

type APIHandler struct {
	auth     *AuthManager
	reg      *registry.Registry
	launcher *launcher.Launcher
	drivers  *driver.Registry
	quotaEng *quota.Engine
}

func NewAPIHandler(auth *AuthManager) *APIHandler {
	return &APIHandler{
		auth:     auth,
		reg:      registry.DefaultRegistry(),
		launcher: launcher.Default(),
		drivers:  driver.DefaultRegistry(),
		quotaEng: quota.NewEngine(5 * time.Minute),
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Health Handler
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "0.4.0",
	})
}

// Session Handler (checks authentication status & returns CSRF token)
func (h *APIHandler) handleSession(w http.ResponseWriter, r *http.Request) {
	sess := h.auth.AuthenticateRequest(r)
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    sess.CSRFToken,
	})
}

// Workspaces / Projects Handler
func (h *APIHandler) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	cfg, _ := config.LoadConfig()

	type WorkspaceView struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Provider string `json:"provider,omitempty"`
		Profile  string `json:"profile,omitempty"`
		IsActive bool   `json:"is_active"`
	}

	var list []WorkspaceView
	// Current workspace
	currentName := cwd
	if idx := strings.LastIndex(cwd, "/"); idx != -1 {
		currentName = cwd[idx+1:]
	}
	list = append(list, WorkspaceView{
		Name:     currentName,
		Path:     cwd,
		IsActive: true,
	})

	// Add workspaces from config bindings
	if len(cfg.Bindings) > 0 {
		for pth, provMap := range cfg.Bindings {
			for prov, prof := range provMap {
				if pth == cwd {
					list[0].Provider = prov
					list[0].Profile = prof
					continue
				}
				wName := pth
				if idx := strings.LastIndex(pth, "/"); idx != -1 {
					wName = pth[idx+1:]
				}
				list = append(list, WorkspaceView{
					Name:     wName,
					Path:     pth,
					Provider: prov,
					Profile:  prof,
					IsActive: false,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, list)
}

// Runtimes List & Start Handler
func (h *APIHandler) handleRuntimes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		_, _ = h.reg.CleanupStale()
		all := h.reg.List()
		writeJSON(w, http.StatusOK, all)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			ProviderID string   `json:"provider"`
			ProfileID  string   `json:"profile"`
			Workspace  string   `json:"workspace"`
			Args       []string `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Workspace == "" {
			req.Workspace, _ = os.Getwd()
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		sess, err := h.launcher.Launch(ctx, launcher.LaunchOptions{
			ProviderID: req.ProviderID,
			ProfileID:  req.ProfileID,
			Workspace:  req.Workspace,
			Args:       req.Args,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, sess)
		return
	}

	if r.Method == http.MethodDelete {
		cleaned, _ := h.reg.CleanupStale()
		purged, _ := h.reg.PurgeInactive()
		writeJSON(w, http.StatusOK, map[string]any{
			"cleaned": cleaned,
			"purged":  purged,
		})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// Runtime Detail, Stop, Handoff, Continue Handlers
func (h *APIHandler) handleRuntimeDetail(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/runtimes/<id>[/<action>]
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/runtimes/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "missing runtime ID")
		return
	}

	runtimeID := parts[0]
	sess, exists := h.reg.Get(runtimeID)
	if !exists {
		writeError(w, http.StatusNotFound, "runtime not found")
		return
	}

	// GET detail
	if len(parts) == 1 && r.Method == http.MethodGet {
		d, _ := h.drivers.Get(sess.ProviderID)
		var effCaps any
		if d != nil {
			effCaps = d.EffectiveCaps(r.Context(), model.Profile{Name: sess.ProfileID, Provider: sess.ProviderID})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"session":      sess,
			"capabilities": effCaps,
		})
		return
	}

	// DELETE runtime record
	if len(parts) == 1 && r.Method == http.MethodDelete {
		_ = h.reg.Delete(runtimeID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	// Actions: stop, handoff, continue
	if len(parts) == 2 && r.Method == http.MethodPost {
		action := parts[1]
		switch action {
		case "stop":
			client, err := protocol.NewClient(runtimeID)
			if err == nil {
				_ = client.Stop()
				_ = client.Close()
			}
			_ = h.reg.UpdateState(runtimeID, registry.StateStopped)
			writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
			return

		case "handoff":
			var payload struct {
				Target string `json:"target"` // provider:profile
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Target == "" {
				writeError(w, http.StatusBadRequest, "missing target profile")
				return
			}
			newSess, err := handoff.PerformAccountHandoff(r.Context(), runtimeID, payload.Target)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, newSess)
			return

		case "continue":
			var payload struct {
				TargetProvider string `json:"provider"`
				TargetProfile  string `json:"profile"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.TargetProvider == "" {
				writeError(w, http.StatusBadRequest, "missing target provider")
				return
			}
			newSess, err := handoff.PerformContextHandoff(r.Context(), runtimeID, payload.TargetProvider, payload.TargetProfile)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, newSess)
			return
		}
	}

	writeError(w, http.StatusBadRequest, "unknown runtime endpoint")
}

// Providers Handler
func (h *APIHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	drivers := h.drivers.List()
	type ProviderView struct {
		ID           string                      `json:"id"`
		Installed    bool                        `json:"installed"`
		Version      string                      `json:"version"`
		ControlLevel registry.ControlLevel       `json:"control_level"`
		Capabilities driver.EffectiveCapabilities `json:"capabilities"`
	}

	var res []ProviderView
	for _, d := range drivers {
		det, _ := d.Detect(r.Context())
		caps := d.EffectiveCaps(r.Context(), model.Profile{Name: "default", Provider: d.ProviderID()})
		res = append(res, ProviderView{
			ID:           d.ProviderID(),
			Installed:    det.Installed,
			Version:      det.Version,
			ControlLevel: caps.ControlLevel,
			Capabilities: caps,
		})
	}
	writeJSON(w, http.StatusOK, res)
}

// Profiles Handler
func (h *APIHandler) handleProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, _ := profile.List()
	writeJSON(w, http.StatusOK, profiles)
}

// Events Handler
func (h *APIHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	runtimeID := r.URL.Query().Get("runtime_id")
	evs := events.DefaultBus().GetHistory(runtimeID, 50)
	if evs == nil {
		evs = []events.Event{}
	}
	writeJSON(w, http.StatusOK, evs)
}
