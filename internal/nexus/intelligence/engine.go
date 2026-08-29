package intelligence

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NexusEngine implements the high-level IntelligenceEngine interface.
type NexusEngine struct {
	provider IntelligenceProvider
}

// NewNexusEngine creates an engine wrapping the given provider or default OpenAI provider.
func NewNexusEngine(p IntelligenceProvider) *NexusEngine {
	if p == nil {
		p = NewOpenAIProvider("", "", "")
	}
	return &NexusEngine{provider: p}
}

func (e *NexusEngine) Analyze(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, error) {
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

// FallbackAnalyzeIntent produces rule-based intent analysis when no LLM API is available.
func FallbackAnalyzeIntent(input string) *IntentAnalysis {
	scope := "mission"
	risk := "low"
	lower := strings.ToLower(input)

	if strings.Contains(lower, "refactor") || strings.Contains(lower, "rewrite") || strings.Contains(lower, "security") {
		risk = "medium"
	}
	if strings.Contains(lower, "database") || strings.Contains(lower, "migrate") || strings.Contains(lower, "delete") {
		risk = "high"
	}

	return &IntentAnalysis{
		Intent:    input,
		Scope:     scope,
		RiskLevel: risk,
		IdentifiedGoals: []string{
			"Implement core functionality described in objective",
			"Validate regression test suite with race detector",
			"Document durable changes in project worklog",
		},
		Constraints: []string{
			"Preserve backward compatibility and existing user configurations",
			"Follow zero-warning TypeScript and Go quality gates",
		},
		Assumptions: []string{
			"Standard Go and Web toolchains are available on host",
		},
		CreatedAt: time.Now().UTC(),
	}
}

// FallbackAmbiguities identifies rule-based ambiguity items.
func FallbackAmbiguities(intent *IntentAnalysis) []AmbiguityItem {
	return []AmbiguityItem{
		{
			Key:              "execution_isolation",
			Level:            AmbiguityImportant,
			Question:         "Deseja executar as alterações diretamente no diretório do projeto ou em um git worktree isolado?",
			Rationale:        "Worktrees isolados evitam conflito com outros agentes ativos no mesmo repositório.",
			SuggestedOptions: []string{"Diretório Canônico do Projeto", "Worktree Isolado"},
			DefaultChoice:    "Diretório Canônico do Projeto",
		},
		{
			Key:              "verification_gate",
			Level:            AmbiguityLowImpact,
			Question:         "Executar suíte completa de testes com detecção de concorrência antes da aprovação?",
			Rationale:        "Garante conformidade com o portão tdd-verification do Orquestrador Maestro.",
			SuggestedOptions: []string{"Sim (go test -race ./...)", "Apenas testes unitários rápidos"},
			DefaultChoice:    "Sim (go test -race ./...)",
		},
	}
}

// FallbackPlanOutline generates structured WorkPackages from the intent.
func FallbackPlanOutline(intent *IntentAnalysis, facts map[string]string) []WorkPackageOutline {
	return []WorkPackageOutline{
		{
			Title:        "Foundation & Core Architecture",
			Goal:         "Establish verified core data models, interfaces, and unit tests.",
			Priority:     "CRITICAL",
			Role:         "implementer",
			Skills:       []string{"skill-refactoring", "skill-verification"},
			Acceptance:   []string{"Unit tests pass with 100% success", "Data models persisted to store"},
			Dependencies: []string{},
		},
		{
			Title:        "Product Integration & REST APIs",
			Goal:         "Implement web endpoints and integrate with Workspace OS controls.",
			Priority:     "HIGH",
			Role:         "implementer",
			Skills:       []string{"skill-saas-factory"},
			Acceptance:   []string{"REST routes registered and tested", "WebSocket state broadcast verified"},
			Dependencies: []string{"Foundation & Core Architecture"},
		},
		{
			Title:        "Autonomous Verification & Quality Gate",
			Goal:         "Execute independent review and full regression test suite.",
			Priority:     "HIGH",
			Role:         "reviewer",
			Skills:       []string{"skill-verification", "skill-security-audit"},
			Acceptance:   []string{"go test -race passes with 0 race conditions", "ESLint passes with 0 errors"},
			Dependencies: []string{"Product Integration & REST APIs"},
		},
	}
}
