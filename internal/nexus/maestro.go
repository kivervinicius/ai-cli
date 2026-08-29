package nexus

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// MaestroVersion is the contract version for Maestro integration.
const MaestroVersion = "1.0.0"

// MaestroMode represents the Maestro integration mode.
type MaestroMode string

const (
	MaestroOff         MaestroMode = "OFF"
	MaestroAssist      MaestroMode = "ASSIST"
	MaestroOrchestrate MaestroMode = "ORCHESTRATE"
)

// MaestroCapability describes what the Maestro instance supports.
type MaestroCapability struct {
	Version     string   `json:"version"`
	Modes       []string `json:"modes"`       // supported modes
	Skills      []string `json:"skills"`      // available skill IDs
	Gates       []string `json:"gates"`       // available gate types
	Processes   []string `json:"processes"`   // available process types
}

// AdviceRequest is the structured request sent to Maestro for recommendations.
type AdviceRequest struct {
	Version   string         `json:"version"`
	Context   AdviceContext  `json:"context"`
	Intent    string         `json:"intent"`
	Scope     string         `json:"scope"`     // "project" | "agent" | "task"
	Extra     map[string]any `json:"extra,omitempty"`
}

// AdviceContext provides project/agent context for Maestro decisions.
type AdviceContext struct {
	ProjectID   string `json:"project_id"`
	AgentID     string `json:"agent_id,omitempty"`
	AgentStatus string `json:"agent_status,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Profile     string `json:"profile,omitempty"`
}

// AdviceResponse is the structured response from Maestro.
type AdviceResponse struct {
	Version    string             `json:"version"`
	Mode       MaestroMode        `json:"mode"`
	Required   []Recommendation   `json:"required"`
	Recommended []Recommendation  `json:"recommended"`
	Optional   []Recommendation   `json:"optional"`
	Explanation string            `json:"explanation,omitempty"`
}

// Recommendation is a single actionable recommendation from Maestro.
type Recommendation struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`        // "action" | "config" | "security" | "process"
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Apply       string         `json:"apply"`       // action identifier
	Why         string         `json:"why"`         // explanation
	Risk        string         `json:"risk"`        // "low" | "medium" | "high"
	Gates       []string       `json:"gates,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
	Verify      string         `json:"verify,omitempty"` // how to verify
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MaestroStatus represents the current Maestro integration state.
type MaestroStatus struct {
	Available    bool              `json:"available"`
	Mode         MaestroMode       `json:"mode"`
	Capabilities *MaestroCapability `json:"capabilities,omitempty"`
	LastCheck    time.Time         `json:"last_check"`
	Error        string            `json:"error,omitempty"`
}

// MaestroClient handles communication with the Maestro process.
type MaestroClient struct {
	status     MaestroStatus
	maestroBin string // path to maestro binary
}

// NewMaestroClient creates a client that discovers the Maestro binary.
func NewMaestroClient() *MaestroClient {
	bin := findMaestroBin()
	c := &MaestroClient{maestroBin: bin}
	c.checkAvailability()
	return c
}

func findMaestroBin() string {
	// Check common paths and binary names.
	candidates := []string{"orquestrador-maestro", "maestro", "orquestrador"}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	// Direct standard paths if not in PATH.
	commonPaths := []string{
		"/home/desenvolvedor/.nvm/versions/node/v22.17.0/bin/orquestrador-maestro",
		"/usr/local/bin/orquestrador-maestro",
		"/usr/local/bin/maestro",
	}
	for _, p := range commonPaths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}

func (c *MaestroClient) checkAvailability() {
	if c.maestroBin == "" {
		c.status = MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     "maestro binary not found",
			LastCheck: time.Now(),
		}
		return
	}

	// Try to get capabilities / version.
	cap, err := c.queryCapabilities()
	if err != nil {
		c.status = MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     err.Error(),
			LastCheck: time.Now(),
		}
		return
	}

	c.status = MaestroStatus{
		Available:    true,
		Mode:         MaestroAssist,
		Capabilities: cap,
		LastCheck:    time.Now(),
	}
}

func (c *MaestroClient) queryCapabilities() (*MaestroCapability, error) {
	if c.maestroBin == "" {
		return nil, fmt.Errorf("no maestro binary")
	}

	// First try standalone 'capabilities --json'
	cmd := exec.Command(c.maestroBin, "capabilities", "--json")
	if out, err := cmd.Output(); err == nil {
		var cap MaestroCapability
		if err := json.Unmarshal(out, &cap); err == nil {
			return &cap, nil
		}
	}

	// Capabilities could not be verified — return error instead of hardcoded data
	return nil, fmt.Errorf("maestro capabilities unverifiable (binary found but capabilities --json failed)")
}

// Status returns the current Maestro integration status.
func (c *MaestroClient) Status() MaestroStatus {
	return c.status
}

// GetAdvice requests recommendations from Maestro for the given context.
func (c *MaestroClient) GetAdvice(ctx AdviceContext, intent string) (*AdviceResponse, error) {
	if !c.status.Available || c.maestroBin == "" {
		return &AdviceResponse{
			Version: MaestroVersion,
			Mode:    MaestroOff,
		}, fmt.Errorf("maestro unavailable (MAESTRO_DEGRADED)")
	}

	// First attempt: standalone advise command if supported
	req := AdviceRequest{
		Version: MaestroVersion,
		Context: ctx,
		Intent:  intent,
		Scope:   "project",
	}
	reqBytes, _ := json.Marshal(req)
	cmdAdvise := exec.Command(c.maestroBin, "advise", "--json")
	cmdAdvise.Stdin = stringToReader(reqBytes)
	if out, err := cmdAdvise.Output(); err == nil {
		var resp AdviceResponse
		if err := json.Unmarshal(out, &resp); err == nil {
			return &resp, nil
		}
	}

	// Bridge mode failed — return degraded response
	return &AdviceResponse{
		Version:     MaestroVersion,
		Mode:        MaestroOff,
		Required:    []Recommendation{},
		Recommended: []Recommendation{},
		Optional:    []Recommendation{},
		Explanation: fmt.Sprintf("Maestro advise command failed for project %s. Binary exists but could not produce recommendations.", ctx.ProjectID),
	}, fmt.Errorf("maestro advise failed (MAESTRO_DEGRADED)")
}

func stringToReader(b []byte) *stringReader {
	return &stringReader{data: b, pos: 0}
}

type stringReader struct {
	data []byte
	pos  int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
