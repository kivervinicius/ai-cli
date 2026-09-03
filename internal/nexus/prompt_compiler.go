package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/maestrogates"
)

type CompiledAgentPrompt struct {
	CompiledPrompt  string   `json:"compiled_prompt"`
	PromptHash      string   `json:"prompt_hash"`
	ValidatedSkills []string `json:"validated_skills"`
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
	}, nil
}
