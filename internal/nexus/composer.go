package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/intelligence"
	"github.com/kivervinicius/ai-cli/internal/nexus/maestrogates"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type ComposerSessionView struct {
	Session         store.ComposerSession            `json:"session"`
	Brief           LivingBrief                      `json:"brief"`
	Turns           []store.ComposerTurn             `json:"turns"`
	Skills          []store.ComposerSkillProposal    `json:"skills"`
	Artifacts       []store.PromptArtifact           `json:"artifacts,omitempty"`
	Variants        map[string][]store.PromptVariant `json:"variants,omitempty"`
	FlowSuitability FlowSuitabilityAssessment        `json:"flow_suitability"`
	PromptReview    ComposerPromptReview             `json:"prompt_review,omitempty"`
}

type FlowSuitability string

const (
	FlowDirectFit           FlowSuitability = "DIRECT_FIT"
	FlowBeneficial          FlowSuitability = "FLOW_BENEFICIAL"
	FlowStronglyRecommended FlowSuitability = "FLOW_STRONGLY_RECOMMENDED"
)

type FlowSuitabilityAssessment struct {
	Result  FlowSuitability `json:"result"`
	Reasons []string        `json:"reasons"`
	Signals []string        `json:"signals"`
}

type ComposerPromptReview struct {
	Missing        []string `json:"missing,omitempty"`
	Contradictions []string `json:"contradictions,omitempty"`
	Diff           []string `json:"diff,omitempty"`
}

func ReviewComposerPrompt(brief LivingBrief) ComposerPromptReview {
	review := ComposerPromptReview{}
	if strings.TrimSpace(brief.SourcePrompt) == "" {
		return review
	}
	if len(brief.Scope.InScope) == 0 {
		review.Missing = append(review.Missing, "scope")
	}
	if len(brief.Quality.AcceptanceCriteria) == 0 {
		review.Missing = append(review.Missing, "acceptance_criteria")
	}
	if len(brief.Quality.Verification) == 0 {
		review.Missing = append(review.Missing, "verification")
	}
	lower := strings.ToLower(brief.SourcePrompt)
	if (strings.Contains(lower, "must ") || strings.Contains(lower, "deve ")) && (strings.Contains(lower, "must not") || strings.Contains(lower, "não deve") || strings.Contains(lower, "nao deve")) {
		review.Contradictions = append(review.Contradictions, "prompt contains both mandatory and prohibitive instructions; confirm precedence")
	}
	if len(review.Missing) > 0 {
		review.Diff = append(review.Diff, "Composer added structured fields: "+strings.Join(review.Missing, ", "))
	}
	return review
}

func AssessComposerFlowSuitability(brief LivingBrief) FlowSuitabilityAssessment {
	signals, reasons := []string{}, []string{}
	if len(brief.Scope.InScope) >= 2 {
		signals = append(signals, "multiple-workstreams")
		reasons = append(reasons, "há múltiplas frentes no escopo")
	}
	if len(brief.Quality.Review) > 0 {
		signals = append(signals, "independent-review")
		reasons = append(reasons, "há revisão adicional")
	}
	if len(brief.Constraints.Security)+len(brief.Risks) > 0 {
		signals = append(signals, "risk-or-gates")
		reasons = append(reasons, "há risco ou restrição que beneficia gates")
	}
	if len(signals) >= 3 {
		return FlowSuitabilityAssessment{FlowStronglyRecommended, reasons, signals}
	}
	if len(signals) > 0 {
		return FlowSuitabilityAssessment{FlowBeneficial, reasons, signals}
	}
	return FlowSuitabilityAssessment{FlowDirectFit, []string{"execução direta é suficiente para este escopo"}, signals}
}

func (n *Nexus) CreateComposerSession(ctx context.Context, projectID, goal string) (*ComposerSessionView, error) {
	return n.CreateComposerSessionWithPrompt(ctx, projectID, goal, "")
}

func (n *Nexus) CreateComposerSessionWithPrompt(ctx context.Context, projectID, goal, sourcePrompt string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	brief := newComposerBrief(goal, sourcePrompt)
	brief.Context.Project = project.Name
	if readiness, err := n.ObserveContextReadiness(projectID); err == nil {
		brief.Context.Evidence = appendUniqueAll(brief.Context.Evidence, []string{
			"Project: " + project.Name,
			"Branch: " + firstNonEmpty(readiness.CurrentFingerprint.Branch, "unknown"),
			"HEAD: " + firstNonEmpty(readiness.CurrentFingerprint.Head, "unknown"),
		})
	}
	refreshComposerBrief(&brief)
	briefJSON, _ := json.Marshal(brief)
	sessionState := composerSessionStateFromBrief(brief)
	session, err := st.CreateComposerSession(store.ComposerSession{
		ProjectID:          projectID,
		Title:              firstNonEmpty(brief.Intent.Objective, brief.Goal),
		State:              sessionState,
		ContextFingerprint: project.CanonicalPath,
		BriefJSON:          string(briefJSON),
	})
	if err != nil {
		return nil, err
	}
	seedComposerSkillProposals(st, session.ID, brief, n.currentMaestroStatus())
	return n.composeSessionView(st, *session, brief)
}

func seedComposerSkillProposals(st *store.Store, sessionID string, brief LivingBrief, status MaestroStatus) {
	if !status.Available || status.Capabilities == nil {
		return
	}
	for _, skill := range status.Capabilities.Skills {
		if !skillRelevantToComposer(skill, brief) {
			continue
		}
		_, _ = st.UpsertComposerSkillProposal(store.ComposerSkillProposal{
			SessionID: sessionID, SkillID: skill.ID, State: store.ComposerSkillSuggested,
			Reason:        "Discovered from the live Maestro capability catalog.",
			Applicability: firstNonEmpty(skill.Description, "Compatible with this Composer brief."),
			Risk:          skill.Risk, Source: "Maestro", Version: firstNonEmpty(skill.Version, status.Capabilities.Version), Available: true,
		})
	}
}

func skillRelevantToComposer(skill MaestroSkillDesc, brief LivingBrief) bool {
	text := strings.ToLower(skill.ID + " " + skill.Name + " " + skill.Description + " " + strings.Join(skill.Triggers, " "))
	if len(brief.Constraints.Security) > 0 || brief.Intent.Archetype == PromptArchetypeSecurity {
		return strings.Contains(text, "secur") || strings.Contains(text, "auth")
	}
	if brief.Intent.Archetype == PromptArchetypeResearch {
		return strings.Contains(text, "research") || strings.Contains(text, "review")
	}
	return true
}

func (n *Nexus) GetComposerSession(ctx context.Context, id string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(id)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	refreshComposerBrief(&brief)
	return n.composeSessionView(st, *session, brief)
}

func (n *Nexus) ListComposerSessions(ctx context.Context, projectID string) ([]store.ComposerSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListComposerSessions(projectID)
}

func (n *Nexus) AddComposerTurn(ctx context.Context, sessionID, role, content string) (*ComposerSessionView, error) {
	return n.AddComposerTurnExpected(ctx, sessionID, role, content, 0)
}

func (n *Nexus) AddComposerTurnExpected(ctx context.Context, sessionID, role, content string, expectedRevision int) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create a new session to keep refining the prompt")
	}
	if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: role, Content: content}); err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	intelligenceNotice := ""
	if role == store.ComposerUser {
		mergeTextIntoBrief(&brief, content, "USER")
		if provider, providerErr := n.ConfiguredIntelligenceProvider(ctx, session.ProjectID); providerErr == nil && provider.Available(ctx) {
			analysis, analyzeErr := provider.AnalyzeIntent(ctx, content, map[string]any{"project_id": session.ProjectID, "brief": brief})
			if analyzeErr != nil {
				intelligenceNotice = "Inteligência local indisponível nesta rodada: " + analyzeErr.Error()
			}
			if analysis != nil {
				brief.Intent.Objective = firstNonEmpty(analysis.Intent, brief.Intent.Objective)
				brief.Intent.DesiredOutcome = firstNonEmpty(analysis.Intent, brief.Intent.DesiredOutcome)
				brief.Constraints.Technical = appendUniqueAll(brief.Constraints.Technical, analysis.Constraints)
				for _, assumption := range analysis.Assumptions {
					brief.Assumptions = upsertAssumption(brief.Assumptions, PromptAssumption{ID: "intelligence-" + fmt.Sprint(len(brief.Assumptions)+1), Value: assumption, Confidence: "MEDIUM", Status: "INFERRED"})
				}
			}
			if analysis != nil {
				unknowns, unknownErr := provider.EvaluateAmbiguities(ctx, analysis)
				if unknownErr != nil {
					intelligenceNotice = "Inteligência local não conseguiu avaliar as lacunas: " + unknownErr.Error()
				}
				for _, unknown := range unknowns {
					brief.OpenQuestions = appendUnique(brief.OpenQuestions, unknown.Question)
				}
			}
		} else if providerErr != nil && !errors.Is(providerErr, intelligence.ErrIntelligenceUnavailable) {
			intelligenceNotice = "Inteligência local indisponível: " + providerErr.Error()
		}
		refreshComposerBrief(&brief)
		response := strings.TrimSpace(strings.Join([]string{intelligenceNotice, composeAssistantReply(brief)}, "\n\n"))
		if strings.TrimSpace(response) != "" {
			if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: store.ComposerAssistant, Content: response}); err != nil {
				return nil, err
			}
		}
	}
	return n.persistComposerSessionExpected(st, session, brief, expectedRevision)
}

func (n *Nexus) UpdateComposerSkillState(ctx context.Context, sessionID, skillID, state string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if err := st.SetComposerSkillState(sessionID, skillID, state); err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	return n.composeSessionView(st, *session, decodeComposerBrief(session.BriefJSON))
}

func (n *Nexus) FinalizeComposerSession(ctx context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	refreshComposerBrief(&brief)
	if brief.Readiness.State == PromptReadinessBlocked && !confirmGaps {
		return nil, fmt.Errorf("composer has blocking gaps; confirm gaps before finalizing")
	}
	if len(brief.OpenQuestions) > 0 && !confirmGaps {
		return nil, fmt.Errorf("composer has open questions; confirm gaps before finalizing")
	}
	skills, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return nil, err
	}
	validatedSkills, err := n.validateComposerSelectedSkills(selectedSkills, skills)
	if err != nil {
		return nil, err
	}
	compiled, err := compileComposerPrompt(renderCanonicalPrompt(brief), validatedSkills, n.currentMaestroStatus())
	if err != nil {
		return nil, err
	}
	contextJSON, _ := json.Marshal(map[string]any{"project_id": session.ProjectID, "brief": brief})
	skillJSON, _ := json.Marshal(compiled.ValidatedSkills)
	artifact, err := st.CreatePromptArtifact(store.PromptArtifact{
		SessionID:    sessionID,
		Content:      compiled.CompiledPrompt,
		ContextJSON:  string(contextJSON),
		SkillIDsJSON: string(skillJSON),
	})
	if err != nil {
		return nil, err
	}
	status := n.currentMaestroStatus()
	var desc []MaestroSkillDesc
	if status.Capabilities != nil {
		for _, candidate := range status.Capabilities.Skills {
			if containsString(validatedSkills, candidate.ID) {
				desc = append(desc, candidate)
			}
		}
	}
	for _, variant := range CompilePromptVariants(brief, desc, "") {
		rawCaps, _ := json.Marshal(map[string]any{"maestro_available": status.Available})
		_, _ = st.CreatePromptVariant(store.PromptVariant{ArtifactID: artifact.ID, Variant: string(variant.Kind), Target: variant.Target, Content: variant.Content, CapabilitiesJSON: string(rawCaps)})
	}
	for _, skill := range skills {
		switch {
		case containsString(validatedSkills, skill.SkillID):
			_ = st.SetComposerSkillState(sessionID, skill.SkillID, store.ComposerSkillApplied)
		case skill.State == store.ComposerSkillApplied:
			_ = st.SetComposerSkillState(sessionID, skill.SkillID, store.ComposerSkillAccepted)
		}
	}
	session.State = store.ComposerFinalized
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		return nil, err
	}
	return artifact, nil
}

// RefineComposerArtifact adds an optional refinement turn to an existing
// composer session and generates a new immutable PromptArtifact revision.
// If refinementGoal is empty the session is re-compiled as-is (useful to
// refresh after resolving unknowns). The session state is reset to allow
// the new finalization pass to proceed even when it was previously finalized.
func (n *Nexus) RefineComposerArtifact(ctx context.Context, sessionID, refinementGoal string) (*store.PromptArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	// Reset finalized state so turns and FinalizeComposerSession do not block.
	if session.State == store.ComposerFinalized {
		session.State = store.ComposerReady
		if err := st.UpdateComposerSession(*session); err != nil {
			return nil, err
		}
	}
	if refinementGoal = strings.TrimSpace(refinementGoal); refinementGoal != "" {
		if _, err := n.AddComposerTurn(ctx, sessionID, store.ComposerUser, "Refinamento: "+refinementGoal); err != nil {
			return nil, err
		}
	}
	// Collect accepted/applied skills to carry forward.
	skills, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return nil, err
	}
	selectedSkillIDs := []string{}
	for _, s := range skills {
		if s.State == store.ComposerSkillAccepted || s.State == store.ComposerSkillApplied {
			selectedSkillIDs = append(selectedSkillIDs, s.SkillID)
		}
	}
	return n.FinalizeComposerSession(ctx, sessionID, selectedSkillIDs, true)
}

// ResolveComposerUnknown updates the status and answer of a single unknown
// item in the living brief and re-evaluates readiness.
func (n *Nexus) ResolveComposerUnknown(_ context.Context, sessionID, unknownID, answer, status string) (*ComposerSessionView, error) {
	return n.ResolveComposerUnknownExpected(context.Background(), sessionID, unknownID, answer, status, 0)
}

func (n *Nexus) ResolveComposerUnknownExpected(ctx context.Context, sessionID, unknownID, answer, status string, expectedRevision int) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	found := false
	for i := range brief.Unknowns {
		if brief.Unknowns[i].ID == unknownID {
			brief.Unknowns[i].Answer = strings.TrimSpace(answer)
			brief.Unknowns[i].Status = PromptUnknownStatus(status)
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown %q not found in session %q", unknownID, sessionID)
	}
	refreshComposerBrief(&brief)
	return n.persistComposerSessionExpected(st, session, brief, expectedRevision)
}

func (n *Nexus) composeSessionView(st *store.Store, session store.ComposerSession, brief LivingBrief) (*ComposerSessionView, error) {
	refreshComposerBrief(&brief)
	turns, err := st.ListComposerTurns(session.ID, 40)
	if err != nil {
		return nil, err
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil {
		return nil, err
	}
	skills = reconcileComposerSkillAvailability(st, session.ID, skills, n.currentMaestroStatus())
	artifacts, err := st.ListPromptArtifacts(session.ID)
	if err != nil {
		return nil, err
	}
	variants := map[string][]store.PromptVariant{}
	for _, artifact := range artifacts {
		if values, variantErr := st.ListPromptVariants(artifact.ID); variantErr == nil && len(values) > 0 {
			variants[artifact.ID] = values
		}
	}
	return &ComposerSessionView{
		Session:         session,
		Brief:           brief,
		Turns:           turns,
		Skills:          skills,
		Artifacts:       artifacts,
		Variants:        variants,
		FlowSuitability: AssessComposerFlowSuitability(brief),
		PromptReview:    ReviewComposerPrompt(brief),
	}, nil
}

func (n *Nexus) persistComposerSessionExpected(st *store.Store, session *store.ComposerSession, brief LivingBrief, expectedRevision int) (*ComposerSessionView, error) {
	session.Title = firstNonEmpty(brief.Intent.Objective, brief.Goal, session.Title)
	session.State = composerSessionStateFromBrief(brief)
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSessionExpected(*session, expectedRevision); err != nil {
		return nil, err
	}
	if expectedRevision > 0 {
		session.Revision = expectedRevision + 1
	} else if latest, getErr := st.GetComposerSession(session.ID); getErr == nil {
		session.Revision = latest.Revision
	}
	return n.composeSessionView(st, *session, brief)
}

func composeAssistantReply(brief LivingBrief) string {
	if len(brief.OpenQuestions) > 0 {
		return "Atualizei o briefing. Próxima pergunta de maior impacto: " + brief.OpenQuestions[0]
	}
	return "Atualizei o briefing. O prompt já está consistente o bastante para finalizar."
}

func decodeComposerBrief(raw string) LivingBrief {
	brief := LivingBrief{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &brief)
	}
	refreshComposerBrief(&brief)
	return brief
}

func composerSessionStateFromBrief(brief LivingBrief) string {
	switch brief.Readiness.State {
	case PromptReadinessReady:
		return store.ComposerReady
	case PromptReadinessReadyWithAssumption:
		return store.ComposerReadyWithGaps
	default:
		return store.ComposerExploring
	}
}

func (n *Nexus) validateComposerSelectedSkills(selected []string, available []store.ComposerSkillProposal) ([]string, error) {
	if len(selected) == 0 {
		return nil, nil
	}
	allowed := map[string]store.ComposerSkillProposal{}
	for _, skill := range available {
		allowed[skill.SkillID] = skill
	}
	filtered := []string{}
	for _, skillID := range selected {
		skillID = strings.TrimSpace(skillID)
		if skillID == "" {
			continue
		}
		item, ok := allowed[skillID]
		if !ok {
			return nil, fmt.Errorf("selected skill %s is not part of this composer session", skillID)
		}
		if item.State == store.ComposerSkillUnavailable {
			return nil, fmt.Errorf("selected skill %s is no longer available", skillID)
		}
		filtered = appendUnique(filtered, skillID)
	}
	return filtered, nil
}

func compileComposerPrompt(userPrompt string, requestedSkills []string, status MaestroStatus) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	validatedSkills := []string{}
	if len(requestedSkills) > 0 {
		catalog := []string{}
		if status.Capabilities != nil {
			catalog = status.Capabilities.SkillIDs()
		}
		var cause error
		if status.Error != "" {
			cause = fmt.Errorf("%s", status.Error)
		}
		var err error
		validatedSkills, err = maestrogates.ValidateStrict(requestedSkills, status.Available, catalog, cause)
		if err != nil {
			return nil, fmt.Errorf("skill validation failed: %w", err)
		}
	}
	return CompileAgentPromptWithValidatedSkills(userPrompt, validatedSkills), nil
}

func reconcileComposerSkillAvailability(st *store.Store, sessionID string, skills []store.ComposerSkillProposal, status MaestroStatus) []store.ComposerSkillProposal {
	if !status.Available || status.Capabilities == nil {
		return skills
	}
	available := map[string]struct{}{}
	for _, skillID := range status.Capabilities.SkillIDs() {
		available[skillID] = struct{}{}
	}
	for i := range skills {
		skills[i].Available = false
		if _, ok := available[skills[i].SkillID]; ok {
			skills[i].Available = true
			continue
		}
		skills[i].State = store.ComposerSkillUnavailable
		_ = st.SetComposerSkillState(sessionID, skills[i].SkillID, store.ComposerSkillUnavailable)
	}
	return skills
}

func writePromptList(b *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n")
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueAll(values []string, additions []string) []string {
	for _, value := range additions {
		values = appendUnique(values, value)
	}
	return values
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func mustJSON(brief LivingBrief) string {
	raw, _ := json.Marshal(brief)
	return string(raw)
}
