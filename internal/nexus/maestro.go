package nexus

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
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

	// Otherwise probe version and router from Orquestrador Maestro CLI
	cmdVer := exec.Command(c.maestroBin, "version")
	outVer, err := cmdVer.Output()
	versionStr := "1.0.0"
	if err == nil {
		versionStr = strings.TrimSpace(string(outVer))
		versionStr = strings.TrimPrefix(versionStr, "Orquestrador Maestro CLI ")
	}

	skills := []string{
		"skill-saas-factory",
		"skill-saas-security-scan",
		"skill-saas-dast-recon",
		"skill-security-hooks",
		"skill-tdd",
		"skill-dev-hierarchy",
	}

	return &MaestroCapability{
		Version:   versionStr,
		Modes:     []string{"ASSIST", "ORCHESTRATE"},
		Skills:    skills,
		Gates:     []string{"WORKLOG_LIMIT", "STRICT_DEV", "GATE_VERIFY"},
		Processes: []string{"observe-route-select-act-verify-report", "compact-context-brief"},
	}, nil
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

	// Bridge mode: Use Orquestrador Maestro protocol rules & skills router
	required := []Recommendation{
		{
			ID:          "maestro-dev-hierarchy",
			Type:        "process",
			Title:       "Project DEV Hierarchy & Canonical Memory",
			Description: "Verify DEV/README.md, DEV/INDEX.md, and update DEV/WORKLOG.md after substantive changes.",
			Apply:       "orquestrador-maestro check-dev-gates",
			Why:         "Enforces cross-tool persistence and prevents session memory loss.",
			Risk:        "low",
			Gates:       []string{"check-dev-gates", "persistence-contract"},
			Skills:      []string{"skill-dev-hierarchy"},
			Verify:      "orquestrador-maestro check-dev-gates --strict",
		},
		{
			ID:          "maestro-verify-gate",
			Type:        "security",
			Title:       "Verification Before Completion",
			Description: "Always run full backend and frontend validation suites before claiming completion.",
			Apply:       "go test ./... && cd web && node node_modules/vitest/dist/cli.js run",
			Why:         "Ensures no silent regressions in build or runtime guarantees.",
			Risk:        "low",
			Gates:       []string{"GATE_VERIFY"},
			Verify:      "go test -race ./...",
		},
	}

	recommended := []Recommendation{
		{
			ID:          "maestro-context-brief",
			Type:        "action",
			Title:       "Dynamic Context Briefing",
			Description: "Generate bounded conversational briefing for current task intent.",
			Apply:       fmt.Sprintf("orquestrador-maestro context brief --task %q --json", intent),
			Why:         "Applies token discipline and prioritizes active specifications.",
			Risk:        "low",
			Skills:      []string{"skill-context-brief"},
		},
		{
			ID:          "maestro-saas-security",
			Type:        "security",
			Title:       "Security & Quality Gates",
			Description: "Apply security scanning and defensive isolation rules to active Agents.",
			Apply:       "orquestrador-maestro doctor",
			Why:         "Protects credentials, workspace boundary and environment tokens.",
			Risk:        "medium",
			Skills:      []string{"skill-saas-security-scan", "skill-security-hooks"},
		},
	}

	return &AdviceResponse{
		Version:     c.status.Capabilities.Version,
		Mode:        MaestroAssist,
		Required:    required,
		Recommended: recommended,
		Optional:    []Recommendation{},
		Explanation: fmt.Sprintf("Maestro Assist actively guiding project %s with %d persistent agents.", ctx.ProjectID, len(required)+len(recommended)),
	}, nil
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
