package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// LivingBrief is the bounded, durable understanding assembled by Composer.
// It intentionally contains conclusions and references, never provider transcripts.
type LivingBrief struct {
	Goal            string   `json:"goal"`
	Context         []string `json:"context,omitempty"`
	Constraints     []string `json:"constraints,omitempty"`
	Decisions       []string `json:"decisions,omitempty"`
	Assumptions     []string `json:"assumptions,omitempty"`
	Alternatives    []string `json:"alternatives,omitempty"`
	Risks           []string `json:"risks,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
	OpenQuestions   []string `json:"open_questions,omitempty"`
}

type ComposerSessionView struct {
	Session   store.ComposerSession         `json:"session"`
	Brief     LivingBrief                   `json:"brief"`
	Turns     []store.ComposerTurn          `json:"turns"`
	Skills    []store.ComposerSkillProposal `json:"skills"`
	Artifacts []store.PromptArtifact        `json:"artifacts,omitempty"`
}

func (n *Nexus) CreateComposerSession(_ context.Context, projectID, goal string) (*ComposerSessionView, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if _, err := st.GetProject(projectID); err != nil {
		return nil, err
	}
	brief := LivingBrief{Goal: strings.TrimSpace(goal)}
	briefJSON, _ := json.Marshal(brief)
	session, err := st.CreateComposerSession(store.ComposerSession{ProjectID: projectID, Title: brief.Goal, State: store.ComposerExploring, BriefJSON: string(briefJSON)})
	if err != nil {
		return nil, err
	}
	return &ComposerSessionView{Session: *session, Brief: brief, Turns: []store.ComposerTurn{}, Skills: []store.ComposerSkillProposal{}}, nil
}

func (n *Nexus) GetComposerSession(_ context.Context, id string) (*ComposerSessionView, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(id)
	if err != nil {
		return nil, err
	}
	brief := LivingBrief{}
	_ = json.Unmarshal([]byte(session.BriefJSON), &brief)
	turns, err := st.ListComposerTurns(id, 40)
	if err != nil {
		return nil, err
	}
	skills, err := st.ListComposerSkillProposals(id)
	if err != nil {
		return nil, err
	}
	return &ComposerSessionView{Session: *session, Brief: brief, Turns: turns, Skills: skills}, nil
}

func (n *Nexus) ListComposerSessions(_ context.Context, projectID string) ([]store.ComposerSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListComposerSessions(projectID)
}

// AddComposerTurn persists a redacted/bounded turn and updates the living brief
// deterministically. Intelligence enrichment is deliberately a later explicit step.
func (n *Nexus) AddComposerTurn(ctx context.Context, sessionID, role, content string) (*ComposerSessionView, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create or resume a draft before adding turns")
	}
	if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: role, Content: content}); err != nil {
		return nil, err
	}
	brief := LivingBrief{}
	_ = json.Unmarshal([]byte(session.BriefJSON), &brief)
	if role == store.ComposerUser && strings.TrimSpace(content) != "" {
		if brief.Goal == "" {
			brief.Goal = strings.TrimSpace(content)
		}
		brief.Context = appendUnique(brief.Context, strings.TrimSpace(content))
	}
	encoded, _ := json.Marshal(brief)
	session.BriefJSON = string(encoded)
	if session.Title == "" {
		session.Title = brief.Goal
	}
	if err := st.UpdateComposerSession(*session); err != nil {
		return nil, err
	}
	if role == store.ComposerUser {
		n.enrichComposerTurn(ctx, st, session, &brief, content)
	}
	return n.GetComposerSession(context.Background(), sessionID)
}

func (n *Nexus) enrichComposerTurn(ctx context.Context, st *store.Store, session *store.ComposerSession, brief *LivingBrief, content string) {
	questions := []string{}
	if provider, err := n.ConfiguredIntelligenceProvider(ctx, session.ProjectID); err == nil && provider.Available(ctx) {
		input := renderCanonicalPrompt(*brief) + "\n\nNew user input:\n" + content
		if intent, analyzeErr := provider.AnalyzeIntent(ctx, input, map[string]any{"composer": true}); analyzeErr == nil && intent != nil {
			brief.Constraints = appendUniqueAll(brief.Constraints, intent.Constraints)
			brief.Assumptions = appendUniqueAll(brief.Assumptions, intent.Assumptions)
			if unknowns, unknownErr := provider.EvaluateAmbiguities(ctx, intent); unknownErr == nil {
				for _, unknown := range unknowns {
					if strings.TrimSpace(unknown.Question) != "" {
						questions = appendUnique(questions, unknown.Question)
					}
				}
			}
		}
	}
	if len(questions) == 0 && len(brief.OpenQuestions) == 0 {
		questions = []string{"Qual resultado mensurável definirá que esta entrega está pronta?", "Quais restrições técnicas, de prazo ou de escopo não podem ser violadas?"}
	}
	brief.OpenQuestions = appendUniqueAll(brief.OpenQuestions, questions)
	encoded, _ := json.Marshal(brief)
	session.BriefJSON = string(encoded)
	if session.Title == "" {
		session.Title = brief.Goal
	}
	_ = st.UpdateComposerSession(*session)
	response := "Registrei o contexto no briefing vivo."
	if len(questions) > 0 {
		response += " Para melhorar o prompt, responda: " + strings.Join(questions, " ")
	}
	_, _ = st.AppendComposerTurn(store.ComposerTurn{SessionID: session.ID, Role: store.ComposerAssistant, Content: response})
	client := NewMaestroClient()
	if advice, err := client.GetAdvice(AdviceContext{ProjectID: session.ProjectID}, brief.Goal); err == nil && advice != nil {
		for _, rec := range append(append(advice.Required, advice.Recommended...), advice.Optional...) {
			for _, skillID := range rec.Skills {
				_, _ = st.UpsertComposerSkillProposal(store.ComposerSkillProposal{SessionID: session.ID, SkillID: skillID, State: store.ComposerSkillSuggested, Reason: rec.Why, Applicability: rec.Title, Risk: rec.Risk})
			}
		}
	}
}

func (n *Nexus) FinalizeComposerSession(_ context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	brief := LivingBrief{}
	_ = json.Unmarshal([]byte(session.BriefJSON), &brief)
	if len(brief.OpenQuestions) > 0 && !confirmGaps {
		return nil, fmt.Errorf("composer has open questions; confirm gaps before finalizing")
	}
	compiled, err := CompileAgentPrompt(renderCanonicalPrompt(brief), selectedSkills, NewMaestroClient())
	if err != nil {
		return nil, err
	}
	skillJSON, _ := json.Marshal(compiled.ValidatedSkills)
	contextJSON, _ := json.Marshal(map[string]any{"project_id": session.ProjectID, "brief": brief})
	artifact, err := st.CreatePromptArtifact(store.PromptArtifact{SessionID: session.ID, Content: compiled.CompiledPrompt, ContextJSON: string(contextJSON), SkillIDsJSON: string(skillJSON)})
	if err != nil {
		return nil, err
	}
	session.State = store.ComposerFinalized
	if err := st.UpdateComposerSession(*session); err != nil {
		return nil, err
	}
	return artifact, nil
}

func renderCanonicalPrompt(brief LivingBrief) string {
	var b strings.Builder
	b.WriteString("# Objective\n")
	b.WriteString(strings.TrimSpace(brief.Goal))
	b.WriteString("\n")
	writePromptList(&b, "Context", brief.Context)
	writePromptList(&b, "Constraints", brief.Constraints)
	writePromptList(&b, "Confirmed decisions", brief.Decisions)
	writePromptList(&b, "Success criteria", brief.SuccessCriteria)
	writePromptList(&b, "Risks", brief.Risks)
	b.WriteString("\nDeliver the objective with evidence for every success criterion. Do not expand scope without asking.\n")
	return strings.TrimSpace(b.String())
}
func writePromptList(b *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString("\n# ")
	b.WriteString(title)
	b.WriteString("\n")
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(value))
			b.WriteString("\n")
		}
	}
}
func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
func appendUniqueAll(values []string, additions []string) []string {
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value != "" {
			values = appendUnique(values, value)
		}
	}
	return values
}
