package intelligence

import (
	"context"
	"time"
)

// AmbiguityLevel classifies unknown requirements based on autonomy boundaries (Phase C).
type AmbiguityLevel string

const (
	AmbiguityBlocking  AmbiguityLevel = "BLOCKING"   // Must ask user before proceeding
	AmbiguityImportant AmbiguityLevel = "IMPORTANT"  // Material scope/risk change
	AmbiguityLowImpact AmbiguityLevel = "LOW_IMPACT" // Sensible default can be chosen autonomously
)

// IntentAnalysis represents structured decomposition of a user or system objective.
type IntentAnalysis struct {
	Intent          string    `json:"intent"`
	Scope           string    `json:"scope"`      // "project" | "mission" | "package" | "task"
	RiskLevel       string    `json:"risk_level"` // "low" | "medium" | "high"
	IdentifiedGoals []string  `json:"identified_goals"`
	Constraints     []string  `json:"constraints"`
	Assumptions     []string  `json:"assumptions"`
	SuggestedStack  []string  `json:"suggested_stack,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AmbiguityItem is an identified unknown that needs classification or user input.
type AmbiguityItem struct {
	Key              string         `json:"key"`
	Level            AmbiguityLevel `json:"level"`
	Question         string         `json:"question"`
	Rationale        string         `json:"rationale"`
	SuggestedOptions []string       `json:"suggested_options,omitempty"`
	DefaultChoice    string         `json:"default_choice,omitempty"`
	Answer           string         `json:"answer,omitempty"`
	IsResolved       bool           `json:"is_resolved"`
}

// ClarificationState tracks unknowns converted into durable facts and constraints.
type ClarificationState struct {
	Unknowns        []AmbiguityItem   `json:"unknowns"`
	StructuredFacts map[string]string `json:"structured_facts"`
	AllBlockingDone bool              `json:"all_blocking_resolved"`
}

// PromptCompilationResult is the compiled, scoped prompt ready for agent execution.
type PromptCompilationResult struct {
	MissionID       string    `json:"mission_id"`
	WorkPackageID   string    `json:"work_package_id"`
	PackageTitle    string    `json:"package_title"`
	SystemPrompt    string    `json:"system_prompt"`
	UserPrompt      string    `json:"user_prompt"`
	MaestroRules    []string  `json:"maestro_rules"`
	AcceptanceGates []string  `json:"acceptance_gates"`
	SharedArtifacts []string  `json:"shared_artifacts"`
	Constraints     []string  `json:"constraints"`
	EstimatedTokens int       `json:"estimated_tokens"`
	CompiledAt      time.Time `json:"compiled_at"`
}

// DecisionExplanation provides transparent reasoning for intelligence routing and plan optimizations.
type DecisionExplanation struct {
	Action     string    `json:"action"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	Timestamp  time.Time `json:"timestamp"`
}

// IntelligenceProvider defines the provider contract (OpenAI-compatible API or CLI).
type IntelligenceProvider interface {
	Name() string
	Available(ctx context.Context) bool
	AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error)
	EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error)
	GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string, contextData map[string]any) ([]WorkPackageOutline, error)
}

// WorkPackageOutline is the raw outline produced by an intelligence provider before optimization.
type WorkPackageOutline struct {
	Title        string   `json:"title"`
	Goal         string   `json:"goal"`
	Priority     string   `json:"priority"` // "CRITICAL" | "HIGH" | "NORMAL" | "LOW"
	Dependencies []string `json:"dependencies"`
	Role         string   `json:"role"`
	Skills       []string `json:"skills"`
	Acceptance   []string `json:"acceptance"`
}

// IntelligenceEngine is the high-level orchestrator for intelligence tasks.
type IntelligenceEngine interface {
	Analyze(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, error)
	ResolveClarification(state *ClarificationState, key string, answer string)
	CompilePrompt(ctx context.Context, pkg WorkPackageOutline, facts map[string]string, maestroSkills []string) (*PromptCompilationResult, error)
}
