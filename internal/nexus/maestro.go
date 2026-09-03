package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	coreconfig "github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/runtime"
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

// MaestroSkillDesc describes a single canonical skill provided by Maestro.
type MaestroSkillDesc struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MaestroCapability describes what the Maestro instance supports.
type MaestroCapability struct {
	Version   string             `json:"version"`
	Modes     []string           `json:"modes"`     // supported modes
	Skills    []MaestroSkillDesc `json:"skills"`    // available skills
	Gates     []string           `json:"gates"`     // available gate types
	Processes []string           `json:"processes"` // available process types
}

func (mc *MaestroCapability) SkillIDs() []string {
	if mc == nil {
		return nil
	}
	ids := make([]string, len(mc.Skills))
	for i, s := range mc.Skills {
		ids[i] = s.ID
	}
	return ids
}

// AdviceRequest is the structured request sent to Maestro for recommendations.
type AdviceRequest struct {
	Version string         `json:"version"`
	Context AdviceContext  `json:"context"`
	Intent  string         `json:"intent"`
	Scope   string         `json:"scope"` // "project" | "agent" | "task"
	Extra   map[string]any `json:"extra,omitempty"`
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
	Version     string           `json:"version"`
	Mode        MaestroMode      `json:"mode"`
	Required    []Recommendation `json:"required"`
	Recommended []Recommendation `json:"recommended"`
	Optional    []Recommendation `json:"optional"`
	Explanation string           `json:"explanation,omitempty"`
	Degraded    bool             `json:"degraded,omitempty"`
}

// Recommendation is a single actionable recommendation from Maestro.
type Recommendation struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // "action" | "config" | "security" | "process"
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Apply       string         `json:"apply"` // action identifier
	Why         string         `json:"why"`   // explanation
	Risk        string         `json:"risk"`  // "low" | "medium" | "high"
	Gates       []string       `json:"gates,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
	Verify      string         `json:"verify,omitempty"` // how to verify
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MaestroStatus represents the current Maestro integration state.
type MaestroStatus struct {
	Available    bool               `json:"available"`
	Mode         MaestroMode        `json:"mode"`
	Capabilities *MaestroCapability `json:"capabilities,omitempty"`
	LastCheck    time.Time          `json:"last_check"`
	Error        string             `json:"error,omitempty"`
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
	// LookPath with enhanced developer toolchains
	candidates := []string{"orquestrador-maestro", "maestro", "orquestrador"}
	for _, name := range candidates {
		if path, err := runtime.LookPath(name); err == nil && path != "" {
			return path
		}
	}

	// Direct paths if not in PATH
	commonPaths := []string{
		"/usr/local/bin/orquestrador-maestro",
		"/usr/local/bin/maestro",
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		commonPaths = append(commonPaths, filepath.Join(home, ".local", "bin", "maestro"))
		commonPaths = append(commonPaths, filepath.Join(home, ".local", "bin", "orquestrador-maestro"))
		matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "orquestrador-maestro"))
		if len(matches) > 0 {
			commonPaths = append(commonPaths, matches[len(matches)-1])
		}
	}
	for _, p := range commonPaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

func findOrquestradorDir() string {
	// Explicit override is useful for CI and non-standard installations.
	if explicit := strings.TrimSpace(os.Getenv("NEXUS_ORQUESTRADOR_DIR")); explicit != "" {
		if fi, err := os.Stat(explicit); err == nil && fi.IsDir() {
			return filepath.Clean(explicit)
		}
	}
	// Host user home.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidate := filepath.Join(home, ".orquestrador")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
	}
	// Isolated ai-cli/Nexus profile homes under the canonical cross-platform DataDir.
	if dataDir, err := coreconfig.DataDir(); err == nil && dataDir != "" {
		pattern := filepath.Join(dataDir, "profiles", "*", "*", "home", ".orquestrador")
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		for _, candidate := range matches {
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				return candidate
			}
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

	// 1. First try standalone 'capabilities --json' if supported in future CLI versions
	cmd := exec.Command(c.maestroBin, "capabilities", "--json")
	if out, err := cmd.Output(); err == nil {
		var cap MaestroCapability
		if err := json.Unmarshal(out, &cap); err == nil && cap.Version != "" {
			return &cap, nil
		}
	}

	// 2. Query version from CLI
	verOut, err := exec.Command(c.maestroBin, "version").Output()
	if err != nil {
		verOut, err = exec.Command(c.maestroBin, "--version").Output()
	}
	version := strings.TrimSpace(string(verOut))
	if version == "" {
		return nil, fmt.Errorf("maestro binary found at %s but failed to report version", c.maestroBin)
	}

	// 3. Read dynamic skills and gates from .orquestrador directory
	orqDir := findOrquestradorDir()
	var skillDescs []MaestroSkillDesc
	if orqDir != "" {
		manifestPath := filepath.Join(orqDir, "SKILLS_MANIFEST.json")
		if data, err := os.ReadFile(manifestPath); err == nil {
			var parsed struct {
				Skills map[string]struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"skills"`
			}
			if err := json.Unmarshal(data, &parsed); err == nil && len(parsed.Skills) > 0 {
				for s, meta := range parsed.Skills {
					name := meta.Name
					if name == "" {
						name = s
					}
					skillDescs = append(skillDescs, MaestroSkillDesc{
						ID:          s,
						Name:        name,
						Description: meta.Description,
					})
				}
			}
		}
		if len(skillDescs) == 0 {
			// Fallback: list skills folder
			skillsDir := filepath.Join(orqDir, "skills")
			if entries, err := os.ReadDir(skillsDir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						id := entry.Name()
						skillDescs = append(skillDescs, MaestroSkillDesc{
							ID:          id,
							Name:        id,
							Description: "",
						})
					}
				}
			}
		}
	}

	sort.Slice(skillDescs, func(i, j int) bool {
		return skillDescs[i].ID < skillDescs[j].ID
	})
	return &MaestroCapability{
		Version: version,
		Modes:   []string{"OFF", "ASSIST", "ORCHESTRATE"},
		Skills:  skillDescs,
		// Gates and processes are intentionally empty unless the Maestro binary
		// reports them through capabilities --json. Nexus never invents them.
		Gates:     []string{},
		Processes: []string{},
	}, nil
}

// Status returns the current Maestro integration status.
func (c *MaestroClient) Status() MaestroStatus {
	return c.status
}

// ListSkills returns all available Maestro skill names.
func (c *MaestroClient) ListSkills(ctx context.Context) ([]string, error) {
	if c.status.Capabilities != nil {
		return c.status.Capabilities.SkillIDs(), nil
	}
	return nil, nil
}

// GetAdvice requests recommendations from Maestro for the given context.
func (c *MaestroClient) GetAdvice(ctx AdviceContext, intent string) (*AdviceResponse, error) {
	if !c.status.Available || c.maestroBin == "" {
		return &AdviceResponse{
			Version:  MaestroVersion,
			Mode:     MaestroOff,
			Degraded: true,
		}, fmt.Errorf("maestro unavailable (MAESTRO_DEGRADED)")
	}

	// 1. Try CLI advise command if supported
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
		if err := json.Unmarshal(out, &resp); err == nil && len(resp.Recommended) > 0 {
			return &resp, nil
		}
	}

	// A failed/malformed advise command means the Maestro contract is unavailable.
	// Do not synthesize skill IDs, gates or process advice inside Nexus.
	version := MaestroVersion
	if c.status.Capabilities != nil && c.status.Capabilities.Version != "" {
		version = c.status.Capabilities.Version
	}
	return &AdviceResponse{
		Version:  version,
		Mode:     MaestroOff,
		Degraded: true,
	}, fmt.Errorf("maestro advise unavailable or returned an invalid contract (MAESTRO_DEGRADED)")

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
