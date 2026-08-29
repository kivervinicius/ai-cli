package intelligence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NexusEngine implements the high-level IntelligenceEngine interface.
type NexusEngine struct {
	provider IntelligenceProvider
}

// ErrIntelligenceUnavailable is returned when an assisted/planned operation is requested without a real provider.
var ErrIntelligenceUnavailable = errors.New("nexus intelligence unavailable")

// NewNexusEngine creates an engine wrapping the explicitly configured provider.
// A nil provider is valid for provider-independent operations such as prompt compilation,
// but Analyze/GeneratePlan fail closed instead of fabricating intelligence output.
func NewNexusEngine(p IntelligenceProvider) *NexusEngine {
	return &NexusEngine{provider: p}
}

func (e *NexusEngine) Analyze(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, error) {
	if e.provider == nil || !e.provider.Available(ctx) {
		return nil, nil, ErrIntelligenceUnavailable
	}
	intent, err := e.provider.AnalyzeIntent(ctx, goal, map[string]any{"project_id": projectID})
	if err != nil {
		return nil, nil, err
	}
	unknowns, err := e.provider.EvaluateAmbiguities(ctx, intent)
	if err != nil {
		return intent, nil, err
	}
	return intent, unknowns, nil
}

// GeneratePlan asks the configured provider for a structured outline. It never
// falls back to locally fabricated work packages.
func (e *NexusEngine) GeneratePlan(ctx context.Context, intent *IntentAnalysis, facts map[string]string) ([]WorkPackageOutline, error) {
	if e.provider == nil || !e.provider.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	packages, err := e.provider.GeneratePlanOutline(ctx, intent, facts)
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("intelligence provider %s returned an empty work plan", e.provider.Name())
	}
	return packages, nil
}

func (e *NexusEngine) ResolveClarification(state *ClarificationState, key string, answer string) {
	if state.StructuredFacts == nil {
		state.StructuredFacts = make(map[string]string)
	}
	state.StructuredFacts[key] = answer

	allDone := true
	for i := range state.Unknowns {
		if state.Unknowns[i].Key == key {
			state.Unknowns[i].Answer = answer
			state.Unknowns[i].IsResolved = true
		}
		if state.Unknowns[i].Level == AmbiguityBlocking && !state.Unknowns[i].IsResolved {
			allDone = false
		}
	}
	state.AllBlockingDone = allDone
}

func (e *NexusEngine) CompilePrompt(
	ctx context.Context,
	pkg WorkPackageOutline,
	facts map[string]string,
	maestroSkills []string,
) (*PromptCompilationResult, error) {
	var factsList []string
	for k, v := range facts {
		factsList = append(factsList, fmt.Sprintf("- **%s**: %s", k, v))
	}

	var skillsList []string
	for _, s := range maestroSkills {
		skillsList = append(skillsList, "- "+s)
	}

	var acceptanceList []string
	for _, a := range pkg.Acceptance {
		acceptanceList = append(acceptanceList, "- [ ] "+a)
	}

	sysPrompt := fmt.Sprintf(`You are an autonomous AI engineering agent executing a structured WorkPackage.
Role: %s
Package: %s
Priority: %s

### Governance & Maestro Rules
%s

### Confirmed Architectural Facts
%s

You must strictly fulfill the acceptance criteria and produce reproducible verification evidence.`,
		pkg.Role, pkg.Title, pkg.Priority, strings.Join(skillsList, "\n"), strings.Join(factsList, "\n"))

	userPrompt := fmt.Sprintf(`## Objective
%s

## Acceptance Criteria
%s

Execute the required changes step-by-step. Validate with automated tests before completing.`,
		pkg.Goal, strings.Join(acceptanceList, "\n"))

	estTokens := (len(sysPrompt) + len(userPrompt)) / 4

	return &PromptCompilationResult{
		PackageTitle:    pkg.Title,
		SystemPrompt:    sysPrompt,
		UserPrompt:      userPrompt,
		MaestroRules:    maestroSkills,
		AcceptanceGates: pkg.Acceptance,
		Constraints:     factsList,
		EstimatedTokens: estTokens,
		CompiledAt:      time.Now().UTC(),
	}, nil
}
