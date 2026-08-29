package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CLIPromptRunner executes one non-interactive prompt using a capability-validated
// provider/profile adapter supplied by the Nexus runtime layer.
type CLIPromptRunner func(ctx context.Context, prompt string) (string, error)

// CLIProvider turns an existing coding-agent CLI into a Nexus Intelligence source.
// The runtime adapter is responsible for validating Headless + SubmitPrompt support
// before setting capabilityValidated=true.
type CLIProvider struct {
	providerID          string
	profile             string
	capabilityValidated bool
	run                 CLIPromptRunner
}

// HeadlessPromptArgs returns the verified non-interactive invocation contract for
// coding-agent CLIs that currently advertise both Headless and SubmitPrompt support.
func HeadlessPromptArgs(providerID, prompt string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "claude", "agy", "gemini", "cursor":
		return []string{"-p", prompt}, nil
	case "opencode":
		return []string{"run", prompt}, nil
	default:
		return nil, fmt.Errorf("provider %q has no verified headless prompt contract", providerID)
	}
}

func NewCLIProvider(providerID, profile string, capabilityValidated bool, run CLIPromptRunner) *CLIProvider {
	return &CLIProvider{providerID: strings.TrimSpace(providerID), profile: strings.TrimSpace(profile), capabilityValidated: capabilityValidated, run: run}
}

func (p *CLIProvider) Name() string {
	if p.profile == "" {
		return "cli:" + p.providerID
	}
	return "cli:" + p.providerID + ":" + p.profile
}

func (p *CLIProvider) Available(context.Context) bool {
	return p != nil && p.providerID != "" && p.profile != "" && p.capabilityValidated && p.run != nil
}

func (p *CLIProvider) AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	ctxJSON, _ := json.Marshal(contextData)
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
You are the Nexus software architecture intelligence layer.
Analyze this engineering objective and produce:
{"intent":"...","scope":"project|mission|package|task","risk_level":"low|medium|high","identified_goals":["..."],"constraints":["..."],"assumptions":["..."]}
Objective: %s
Context: %s`, input, string(ctxJSON))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s analyze intent: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s analyze intent: %w", p.Name(), err)
	}
	var result IntentAnalysis
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s analyze intent: %w", p.Name(), err)
	}
	result.CreatedAt = time.Now().UTC()
	return &result, nil
}

func (p *CLIProvider) EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	intentJSON, _ := json.Marshal(intent)
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
Identify requirement ambiguities. BLOCKING means execution cannot safely continue; IMPORTANT materially changes design; LOW_IMPACT has a safe default.
Schema: {"unknowns":[{"key":"...","level":"BLOCKING|IMPORTANT|LOW_IMPACT","question":"...","rationale":"...","suggested_options":["..."],"default_choice":"..."}]}
Intent: %s`, string(intentJSON))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s ambiguity analysis: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s ambiguity analysis: %w", p.Name(), err)
	}
	var result struct {
		Unknowns []AmbiguityItem `json:"unknowns"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s ambiguity analysis: %w", p.Name(), err)
	}
	return result.Unknowns, nil
}

func (p *CLIProvider) GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string) ([]WorkPackageOutline, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	input, _ := json.Marshal(map[string]any{"intent": intent, "confirmed_facts": facts})
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
Decompose the engineering objective into independently reviewable WorkPackages. Do NOT invent Maestro skill identifiers. The skills array must be empty; Maestro is a separate authority.
Schema: {"packages":[{"title":"...","goal":"...","priority":"CRITICAL|HIGH|NORMAL|LOW","dependencies":["package title"],"role":"implementer|reviewer|tester|architect","skills":[],"acceptance":["measurable criterion"]}]}
Input: %s`, string(input))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s plan generation: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s plan generation: %w", p.Name(), err)
	}
	var result struct {
		Packages []WorkPackageOutline `json:"packages"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s plan generation: %w", p.Name(), err)
	}
	for i := range result.Packages {
		result.Packages[i].Skills = nil
	}
	return result.Packages, nil
}

func extractJSONObject(out string) ([]byte, error) {
	trimmed := strings.TrimSpace(out)
	if strings.HasPrefix(trimmed, "```") {
		if nl := strings.Index(trimmed, "\n"); nl >= 0 {
			trimmed = trimmed[nl+1:]
		}
		if end := strings.LastIndex(trimmed, "```"); end >= 0 {
			trimmed = trimmed[:end]
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON object in provider output")
	}
	return []byte(trimmed[start : end+1]), nil
}
