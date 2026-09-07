package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/maestrogates"
)

type PromptVariantKind string

const (
	PromptVariantGenericPortable PromptVariantKind = "GENERIC_PORTABLE"
	PromptVariantNexusAgent      PromptVariantKind = "NEXUS_AGENT"
	PromptVariantFlowHandoff     PromptVariantKind = "FLOW_HANDOFF"
)

type CompiledPromptVariant struct {
	Kind         PromptVariantKind `json:"kind"`
	Target       string            `json:"target,omitempty"`
	Content      string            `json:"content"`
	Hash         string            `json:"hash"`
	Capabilities []string          `json:"capabilities,omitempty"`
}

func CompilePromptVariants(brief LivingBrief, skills []MaestroSkillDesc, target string) []CompiledPromptVariant {
	base := renderCanonicalPrompt(brief)
	manifest := "\n\nSkill provenance manifest:\n"
	if len(skills) == 0 {
		manifest += "- No externally validated skills selected.\n"
	} else {
		for _, skill := range skills {
			manifest += fmt.Sprintf("- %s (source: Maestro, version: %s)\n", skill.ID, firstNonEmpty(skill.Version, "unknown"))
		}
	}
	variants := []CompiledPromptVariant{}
	for _, kind := range []PromptVariantKind{PromptVariantGenericPortable, PromptVariantNexusAgent, PromptVariantFlowHandoff} {
		content := base + manifest
		if kind == PromptVariantNexusAgent {
			content += "\n\nExecution target: Nexus Agent " + strings.TrimSpace(target) + ". Respect Nexus policies and report evidence."
		}
		if kind == PromptVariantFlowHandoff {
			content += "\n\nHandoff target: Flow draft only. Preserve Composer lineage; do not start execution from this prompt."
		}
		sum := sha256.Sum256([]byte(content))
		variants = append(variants, CompiledPromptVariant{Kind: kind, Target: target, Content: content, Hash: hex.EncodeToString(sum[:])})
	}
	return variants
}

type CompiledAgentPrompt struct {
	CompiledPrompt  string   `json:"compiled_prompt"`
	PromptHash      string   `json:"prompt_hash"`
	ValidatedSkills []string `json:"validated_skills"`
}

// CompileAgentPromptWithValidatedSkills constructs the compiled prompt envelope with already validated skills.
func CompileAgentPromptWithValidatedSkills(userPrompt string, validatedSkills []string) *CompiledAgentPrompt {
	var compiled string
	if len(validatedSkills) == 0 {
		compiled = userPrompt
	} else {
		var b strings.Builder
		b.WriteString("Nexus execution context\n")
		b.WriteString("Scope: next prompt only\n")
		b.WriteString("Validated Maestro skills:\n")
		for _, s := range validatedSkills {
			b.WriteString("- " + s + "\n")
		}
		b.WriteString("\nUser request:\n")
		b.WriteString(userPrompt)
		compiled = b.String()
	}

	h := sha256.Sum256([]byte(compiled))
	hashStr := hex.EncodeToString(h[:])

	return &CompiledAgentPrompt{
		CompiledPrompt:  compiled,
		PromptHash:      hashStr,
		ValidatedSkills: validatedSkills,
	}
}

// CompileAgentPrompt compiles user request and optional Maestro skill ids into an
// honest execution envelope.
func CompileAgentPrompt(userPrompt string, requestedSkills []string, maestroClient *MaestroClient) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var validatedSkills []string
	if len(requestedSkills) > 0 {
		if maestroClient == nil {
			maestroClient = NewMaestroClient()
		}
		status := maestroClient.Status()
		var cause error
		if status.Error != "" {
			cause = fmt.Errorf("%s", status.Error)
		}
		var catalog []string
		if status.Capabilities != nil {
			catalog = status.Capabilities.SkillIDs()
		}
		var err error
		validatedSkills, err = maestrogates.ValidateStrict(requestedSkills, status.Available, catalog, cause)
		if err != nil {
			return nil, fmt.Errorf("skill validation failed: %w", err)
		}
	}

	return CompileAgentPromptWithValidatedSkills(userPrompt, validatedSkills), nil
}
