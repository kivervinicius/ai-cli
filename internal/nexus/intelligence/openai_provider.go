package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAIIntelligenceProvider connects to OpenAI, DeepSeek, Ollama or any compatible /v1/chat/completions endpoint.
type OpenAIIntelligenceProvider struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewOpenAIProvider creates an intelligence provider from environment or explicit config.
func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIIntelligenceProvider {
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("DEEPSEEK_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		}
	}

	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = os.Getenv("DEEPSEEK_MODEL")
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	return &OpenAIIntelligenceProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p *OpenAIIntelligenceProvider) Name() string {
	return "openai-compatible (" + p.Model + ")"
}

func (p *OpenAIIntelligenceProvider) Available(ctx context.Context) bool {
	return p.APIKey != "" || strings.Contains(p.BaseURL, "localhost") || strings.Contains(p.BaseURL, "127.0.0.1")
}

func (p *OpenAIIntelligenceProvider) AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error) {
	if !p.Available(ctx) {
		// Rule-based fallback if no API key is set
		return FallbackAnalyzeIntent(input), nil
	}

	sysPrompt := `You are an expert software engineering architect. Analyze the user's objective and output ONLY valid JSON matching this schema:
{
  "intent": "summary of intent",
  "scope": "project" | "mission" | "package" | "task",
  "risk_level": "low" | "medium" | "high",
  "identified_goals": ["goal 1", "goal 2"],
  "constraints": ["constraint 1"],
  "assumptions": ["assumption 1"]
}`

	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": input},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return FallbackAnalyzeIntent(input), nil
	}

	var parsed IntentAnalysis
	if err := json.Unmarshal(resBytes, &parsed); err != nil {
		return FallbackAnalyzeIntent(input), nil
	}
	parsed.CreatedAt = time.Now().UTC()
	return &parsed, nil
}

func (p *OpenAIIntelligenceProvider) EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error) {
	if !p.Available(ctx) {
		return FallbackAmbiguities(intent), nil
	}

	sysPrompt := `Identify critical unknowns or architectural forks for the given intent.
Classify each item strictly as BLOCKING (user must answer before coding), IMPORTANT (design preference), or LOW_IMPACT (safe default).
Output ONLY valid JSON matching:
{
  "unknowns": [
    {
      "key": "unique_key",
      "level": "BLOCKING" | "IMPORTANT" | "LOW_IMPACT",
      "question": "clear concise question",
      "rationale": "why this matters",
      "suggested_options": ["Option A", "Option B"],
      "default_choice": "Option A"
    }
  ]
}`

	intentBytes, _ := json.Marshal(intent)
	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(intentBytes)},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return FallbackAmbiguities(intent), nil
	}

	var wrapper struct {
		Unknowns []AmbiguityItem `json:"unknowns"`
	}
	if err := json.Unmarshal(resBytes, &wrapper); err != nil {
		return FallbackAmbiguities(intent), nil
	}
	return wrapper.Unknowns, nil
}

func (p *OpenAIIntelligenceProvider) GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string) ([]WorkPackageOutline, error) {
	if !p.Available(ctx) {
		return FallbackPlanOutline(intent, facts), nil
	}

	sysPrompt := `Decompose the intent into a series of structured, reviewable WorkPackages.
Each package must have clear acceptance criteria and required skills.
Output ONLY valid JSON matching:
{
  "packages": [
    {
      "title": "Package Title",
      "goal": "Specific measurable objective",
      "priority": "CRITICAL" | "HIGH" | "NORMAL" | "LOW",
      "dependencies": [],
      "role": "implementer" | "reviewer" | "tester" | "architect",
      "skills": ["skill-refactoring", "skill-verification"],
      "acceptance": ["criterion 1", "test passes"]
    }
  ]
}`

	userContent := map[string]any{
		"intent":          intent,
		"confirmed_facts": facts,
	}
	userBytes, _ := json.Marshal(userContent)

	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(userBytes)},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return FallbackPlanOutline(intent, facts), nil
	}

	var wrapper struct {
		Packages []WorkPackageOutline `json:"packages"`
	}
	if err := json.Unmarshal(resBytes, &wrapper); err != nil {
		return FallbackPlanOutline(intent, facts), nil
	}
	return wrapper.Packages, nil
}

func (p *OpenAIIntelligenceProvider) doRequest(ctx context.Context, body map[string]any) ([]byte, error) {
	reqData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := p.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBytes, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("malformed chat response")
	}

	return []byte(chatResp.Choices[0].Message.Content), nil
}
