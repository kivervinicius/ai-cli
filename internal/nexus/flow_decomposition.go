package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// FlowDecompositionRequest defines the contract for decomposing a prompt or artifact into a Flow proposal (PLAN 04).
type FlowDecompositionRequest struct {
	ProjectID     string   `json:"project_id"`
	ArtifactID    string   `json:"artifact_id,omitempty"`
	Goal          string   `json:"goal"`
	SourcePrompt  string   `json:"source_prompt,omitempty"`
	MaestroSkills []string `json:"maestro_skills,omitempty"`
	Simple        bool     `json:"simple,omitempty"`
}

// FlowDecompositionProposal represents an inspectable candidate flow DAG before final persistence (PLAN 04).
type FlowDecompositionProposal struct {
	Title                  string         `json:"title"`
	Description            string         `json:"description"`
	Archetype              string         `json:"archetype"`
	SourceArtifactID       string         `json:"source_artifact_id,omitempty"`
	SourceArtifactRevision int            `json:"source_artifact_revision,omitempty"`
	Flow                   FlowDefinition `json:"flow"`
	Reasoning              string         `json:"reasoning"`
	MaestroAdvice          string         `json:"maestro_advice,omitempty"`
}

// FlowPreflightCheck represents a single preflight verification item (PLAN 05).
type FlowPreflightCheck struct {
	Key     string `json:"key"` // "dag_validity", "agent_allocation", "resources", "worktree_isolation", "security"
	Label   string `json:"label"`
	Status  string `json:"status"` // "PASS", "WARN", "FAIL"
	Summary string `json:"summary"`
}

// FlowPreflightReport represents the full preflight report before starting a run (PLAN 05).
type FlowPreflightReport struct {
	PlanID      string               `json:"plan_id"`
	Revision    int                  `json:"revision"`
	Strict      bool                 `json:"strict,omitempty"`
	Ready       bool                 `json:"ready"`
	Checks      []FlowPreflightCheck `json:"checks"`
	GeneratedAt time.Time            `json:"generated_at"`
}

// DecomposePromptIntoFlowProposal creates an intelligent, honest DAG proposal from a prompt or goal.
// If the goal represents an atomic, simple task, it produces a clean 1-step Flow without artificial bloat.
func (n *Nexus) DecomposePromptIntoFlowProposal(ctx context.Context, req FlowDecompositionRequest) (*FlowDecompositionProposal, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	project, err := st.GetProject(req.ProjectID)
	if err != nil {
		return nil, err
	}
	goal := strings.TrimSpace(req.Goal)
	sourcePrompt := strings.TrimSpace(req.SourcePrompt)
	var sourceArtifactRevision int
	if req.ArtifactID != "" {
		artifact, err := st.GetPromptArtifact(req.ArtifactID)
		if err != nil {
			return nil, fmt.Errorf("resolve source prompt artifact: %w", err)
		}
		session, err := st.GetComposerSession(artifact.SessionID)
		if err != nil {
			return nil, fmt.Errorf("resolve source composer session: %w", err)
		}
		if session.ProjectID != req.ProjectID {
			return nil, fmt.Errorf("source prompt artifact belongs to another project")
		}
		sourceArtifactRevision = artifact.Version
		if sourcePrompt == "" {
			sourcePrompt = strings.TrimSpace(artifact.Content)
		}
		if goal == "" {
			goal = strings.TrimSpace(artifact.Content)
		}
	}

	archetype := string(classifyPromptArchetype(goal + "\n" + sourcePrompt))

	// Determine if simple
	lowerGoal := strings.ToLower(goal + " " + sourcePrompt)
	isAtomic := req.Simple || strings.Contains(lowerGoal, "fix typo") || strings.Contains(lowerGoal, "ajuste de digitação") ||
		strings.Contains(lowerGoal, "quick fix") || strings.Contains(lowerGoal, "simple fix") ||
		(len(strings.Fields(goal)) <= 5 && !strings.Contains(lowerGoal, "refactor") && !strings.Contains(lowerGoal, "e2e") && !strings.Contains(lowerGoal, "system"))

	title := firstNonEmpty(goal, "Flow Proposal")
	verification := projectVerificationCommands(project.CanonicalPath)
	phaseID := "phase_" + ids.NewRuntimeID()
	flow := FlowDefinition{
		ID:          "flow_draft_" + ids.NewRuntimeID(),
		ProjectID:   req.ProjectID,
		Title:       title,
		Description: fmt.Sprintf("Proposta de decomposição inteligente para: %s", title),
		Status:      "DRAFT",
		Revision:    1,
		Policy:      FlowGuided,
		Phases: []FlowPhase{
			{ID: phaseID, Title: "Execução", Order: 1},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if isAtomic {
		flow.Steps = []FlowStep{
			{
				ID:                       "step_" + ids.NewRuntimeID(),
				PhaseID:                  phaseID,
				Order:                    1,
				Title:                    title,
				Goal:                     firstNonEmpty(sourcePrompt, goal),
				Priority:                 "HIGH",
				Status:                   "READY",
				Role:                     "implementer",
				AssignmentStrategy:       FlowAssignmentAuto,
				AcceptanceCriteria:       []string{"Ajuste concluído com validação dos testes"},
				VerificationRequirements: verification,
				MaestroSkills:            req.MaestroSkills,
				CompiledPrompt:           sourcePrompt,
			},
		}
	} else {
		// Multi-step structured decomposition (Implementer -> Tester/Reviewer)
		step1ID := "step_" + ids.NewRuntimeID()
		step2ID := "step_" + ids.NewRuntimeID()
		flow.Steps = []FlowStep{
			{
				ID:                       step1ID,
				PhaseID:                  phaseID,
				Order:                    1,
				Title:                    "Implementação · " + title,
				Goal:                     fmt.Sprintf("Executar implementação conforme objetivo: %s", goal),
				Priority:                 "HIGH",
				Status:                   "READY",
				Role:                     "implementer",
				AssignmentStrategy:       FlowAssignmentAuto,
				AcceptanceCriteria:       []string{"Funcionalidade implementada sem regressões de build"},
				VerificationRequirements: verification,
				MaestroSkills:            req.MaestroSkills,
				CompiledPrompt:           sourcePrompt,
			},
			{
				ID:                       step2ID,
				PhaseID:                  phaseID,
				Order:                    2,
				Title:                    "Verificação & Testes · " + title,
				Goal:                     "Verificar comportamento e suite de testes",
				Priority:                 "HIGH",
				Status:                   "PENDING",
				Role:                     "reviewer",
				AssignmentStrategy:       FlowAssignmentAuto,
				Dependencies:             []string{step1ID},
				AcceptanceCriteria:       []string{"Todos os testes passando e suite de qualidade aprovada"},
				VerificationRequirements: verification,
			},
		}
	}

	proposal := &FlowDecompositionProposal{
		Title:                  title,
		Description:            flow.Description,
		Archetype:              archetype,
		SourceArtifactID:       req.ArtifactID,
		SourceArtifactRevision: sourceArtifactRevision,
		Flow:                   flow,
		Reasoning:              fmt.Sprintf("Decomposição baseada no arquétipo %s com %d passos.", archetype, len(flow.Steps)),
	}
	return proposal, nil
}

func projectVerificationCommands(root string) []string {
	commands := make([]string, 0, 2)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		commands = append(commands, "go test ./...")
	}
	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		commands = append(commands, "npm test")
	}
	return commands
}

// PreflightFlow validates a Flow or WorkPlan before execution, generating an honest readiness report (PLAN 05).
func (n *Nexus) PreflightFlow(ctx context.Context, planID string) (*FlowPreflightReport, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}

	flow := FlowFromWorkPlan(*plan)
	report := &FlowPreflightReport{
		PlanID:      planID,
		Revision:    plan.CurrentRevision,
		Ready:       true,
		GeneratedAt: time.Now().UTC(),
		Checks:      []FlowPreflightCheck{},
	}

	// 1. DAG Validity
	if err := ValidateFlowDAG(flow); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "dag_validity",
			Label:   "Validação do Grafo (DAG)",
			Status:  "FAIL",
			Summary: fmt.Sprintf("Ciclo ou dependência inválida detectada: %s", err.Error()),
		})
	} else {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "dag_validity",
			Label:   "Validação do Grafo (DAG)",
			Status:  "PASS",
			Summary: fmt.Sprintf("Topologia válida sem ciclos (%d passos ordenados).", len(flow.Steps)),
		})
	}

	// 2. Resource availability check
	resources, resErr := n.ListResources()
	if resErr != nil || len(resources) == 0 {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "resources",
			Label:   "Recursos & Provedores",
			Status:  "WARN",
			Summary: "Nenhum recurso de provedor autenticado detectado. Execução dependerá do ambiente padrão.",
		})
	} else {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "resources",
			Label:   "Recursos & Provedores",
			Status:  "PASS",
			Summary: fmt.Sprintf("%d provedores/perfis disponíveis no ambiente.", len(resources)),
		})
	}

	// 3. Worktree isolation check
	report.Checks = append(report.Checks, FlowPreflightCheck{
		Key:     "worktree_isolation",
		Label:   "Isolamento de Worktree Git",
		Status:  "WARN",
		Summary: "O preflight não consegue provar isolamento por step; a execução autônoma exige configuração explícita de worktree.",
	})

	// 4. Security & Autonomy permissions
	report.Checks = append(report.Checks, FlowPreflightCheck{
		Key:     "security",
		Label:   "Contrato de Autonomia & Segurança",
		Status:  "WARN",
		Summary: "O preflight confirma apenas o contrato declarado; enforcement adicional depende do runtime e não é simulado como PASS.",
	})

	return report, nil
}

// PreflightFlowStrict performs the admission checks required before an
// autonomous mission may dispatch work. PreflightFlow remains advisory for
// legacy callers and intentionally keeps WARN checks non-blocking.
func (n *Nexus) PreflightFlowStrict(ctx context.Context, planID string, autonomous bool) (*FlowPreflightReport, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	report, err := n.PreflightFlow(ctx, planID)
	if err != nil {
		return nil, err
	}
	report.Strict = true
	return n.strictAdmissionForPlan(ctx, *plan, report, autonomous)
}

func (n *Nexus) strictAdmissionForPlan(ctx context.Context, plan store.WorkPlan, report *FlowPreflightReport, autonomous bool) (*FlowPreflightReport, error) {
	if err := validateFlowExecutionContract(plan); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "execution_contract", Label: "Contrato de Execução", Status: "FAIL", Summary: err.Error()})
	}
	project, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	projectRecord, err := project.GetProject(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	if autonomous && strings.ToLower(strings.TrimSpace(projectRecord.DefaultIsolation)) != "worktree" {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "worktree_isolation", Label: "Isolamento de Worktree Git", Status: "FAIL", Summary: "execução autônoma exige DefaultIsolation=worktree no projeto"})
	}
	resources, resourceErr := n.ListResources()
	if resourceErr != nil || len(resources) == 0 {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "resources", Label: "Recursos & Provedores", Status: "FAIL", Summary: "nenhum recurso de provedor disponível para admissão strict"})
	} else {
		for _, phase := range plan.Phases {
			for _, pkg := range phase.Packages {
				if strings.TrimSpace(pkg.Provider) == "" && strings.TrimSpace(pkg.Profile) == "" {
					continue
				}
				if len(filterFlowResourceAccounts(resources, pkg.Provider, pkg.Profile)) == 0 {
					report.Ready = false
					report.Checks = append(report.Checks, FlowPreflightCheck{Key: "resources", Label: "Recursos & Provedores", Status: "FAIL", Summary: fmt.Sprintf("nenhum recurso satisfaz as restrições do step %s", pkg.ID)})
				}
			}
		}
	}
	requested := make([]string, 0)
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			requested = append(requested, pkg.MaestroGates...)
			requested = append(requested, pkg.MaestroSkills...)
		}
	}
	if _, err := n.validateMaestroGatesStrict(uniqueStrings(requested)); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "maestro", Label: "Gates Maestro", Status: "FAIL", Summary: err.Error()})
	}
	return report, nil
}
