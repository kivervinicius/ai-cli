package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/autonomyguard"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type nexusPackageExecutor struct {
	n *Nexus
}

func newNexusPackageExecutor(n *Nexus) runner.PackageExecutor {
	return &nexusPackageExecutor{n: n}
}

func (e *nexusPackageExecutor) Allocate(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.AllocationResult, error) {
	if e.n == nil {
		return runner.AllocationResult{}, fmt.Errorf("nexus executor unavailable")
	}
	st, err := e.n.OpenProject()
	if err != nil {
		return runner.AllocationResult{}, err
	}

	var agent store.Agent
	if strings.TrimSpace(pkg.AssignedAgent) != "" {
		agent, err = st.GetAgent(pkg.AssignedAgent, run.ProjectID)
		if err != nil {
			return runner.AllocationResult{}, fmt.Errorf("assigned agent %s is unavailable: %w", pkg.AssignedAgent, err)
		}
	} else {
		name := strings.TrimSpace(pkg.Title)
		if name == "" {
			name = pkg.PackageID
		}
		agent, err = st.CreateAgent(store.Agent{
			ProjectID: run.ProjectID,
			Name:      "Mission · " + name,
			Role:      defaultRole(pkg.Role, "implementer"),
		})
		if err != nil {
			return runner.AllocationResult{}, fmt.Errorf("create persistent mission agent: %w", err)
		}
	}

	req := missionTaskRequirements(pkg)
	accounts, err := e.n.ListResources()
	if err != nil {
		return runner.AllocationResult{}, err
	}
	if len(accounts) == 0 {
		return runner.AllocationResult{}, fmt.Errorf("no configured provider profiles are available for autonomous execution")
	}

	current, _ := currentAgentConfig(st, agent)
	selected, keepCurrent := selectCurrentResource(accounts, current, req)
	if !keepCurrent {
		recommendation := RecommendResources(accounts, req, PolicyBalanced)
		if recommendation.Recommended == nil {
			return runner.AllocationResult{}, fmt.Errorf("no provider/profile satisfies mission requirements: %s", recommendation.Explanation)
		}
		selected = recommendation.Recommended.Account
	}

	current.Provider = selected.Provider
	current.Profile = selected.Profile
	// Autonomous implementers always use a dedicated worktree unless the user
	// explicitly pinned a workspace in the Agent configuration.
	if strings.TrimSpace(current.Workspace) == "" {
		current.Isolation = "worktree"
	}
	if _, err := e.n.SafeApply(ctx, agent.ID, current); err != nil {
		return runner.AllocationResult{}, fmt.Errorf("persist mission resource allocation: %w", err)
	}

	project, err := st.GetProject(run.ProjectID)
	if err != nil {
		return runner.AllocationResult{}, err
	}
	workspace, err := e.n.resolveExecutionWorkspace(ctx, project, agent, current)
	if err != nil {
		return runner.AllocationResult{}, err
	}
	return runner.AllocationResult{AgentID: agent.ID, Workspace: workspace}, nil
}

func (e *nexusPackageExecutor) Compile(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.PromptArtifact, error) {
	snapshot, err := e.n.loadMissionSnapshot(run)
	if err != nil {
		return runner.PromptArtifact{}, err
	}
	compiled, err := compilePackagePromptFromExecutionSnapshot(ctx, &snapshot.Plan, pkg.PhaseID, pkg.PackageID)
	if err != nil {
		return runner.PromptArtifact{}, err
	}
	content := strings.TrimSpace(compiled.SystemPrompt) + "\n\n" + strings.TrimSpace(compiled.UserPrompt)
	if strings.TrimSpace(content) == "" {
		return runner.PromptArtifact{}, fmt.Errorf("compiled prompt is empty")
	}
	if len(compiled.AcceptanceGates) > 0 {
		content += "\n\n## Acceptance gates\n- " + strings.Join(compiled.AcceptanceGates, "\n- ")
	}
	if strings.TrimSpace(pkg.RemediationContext) != "" {
		content += "\n\n## Remediation evidence from the previous attempt\n" + strings.TrimSpace(pkg.RemediationContext) +
			"\n\nAddress the evidence above. Do not repeat the same failed approach without a concrete change."
	}
	content += autonomyPromptBoundaries(run.Contract)

	h := sha256.Sum256([]byte(content))
	created, err := e.n.st.CreatePromptVersion(store.PromptVersion{
		PlanID:       run.PlanID,
		PackageID:    pkg.PackageID,
		PlanRevision: run.PlanRevision,
		ContentHash:  hex.EncodeToString(h[:]),
		Content:      content,
	})
	if err != nil {
		return runner.PromptArtifact{}, fmt.Errorf("persist immutable prompt version: %w", err)
	}
	return runner.PromptArtifact{VersionID: created.ID, Content: content}, nil
}

func autonomyPromptBoundaries(contract runner.AutonomyContract) string {
	var rules []string
	if contract.DisallowDestructiveGit {
		rules = append(rules, "Do not use destructive git operations such as reset --hard, force push, or deleting branches.")
	}
	if !contract.AllowGitPush {
		rules = append(rules, "Do not push commits or branches to any remote.")
	}
	if !contract.AllowDeploy {
		rules = append(rules, "Do not deploy, publish releases, or modify external production systems.")
	}
	if len(contract.AllowedFilePatterns) > 0 {
		rules = append(rules, "Only modify files matching: "+strings.Join(contract.AllowedFilePatterns, ", "))
	}
	if len(rules) == 0 {
		return ""
	}
	return "\n\n## Autonomy boundaries\n- " + strings.Join(rules, "\n- ")
}

func (e *nexusPackageExecutor) Execute(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, prompt string) (runner.ExecutionResult, error) {
	if strings.TrimSpace(pkg.AssignedAgent) == "" {
		return runner.ExecutionResult{}, fmt.Errorf("package has no assigned persistent agent")
	}
	result, err := e.n.executeAgentPrompt(ctx, pkg.AssignedAgent, pkg.Workspace, prompt, agentPromptPolicy{Contract: run.Contract})
	if err != nil {
		return runner.ExecutionResult{}, err
	}
	if len(run.Contract.AllowedFilePatterns) > 0 {
		changed, guardErr := autonomyguard.GitChangedPaths(ctx, pkg.Workspace)
		if guardErr != nil {
			return runner.ExecutionResult{}, fmt.Errorf("enforce allowed file patterns: %w", guardErr)
		}
		if guardErr := autonomyguard.ValidateAllowedChanges(changed, run.Contract.AllowedFilePatterns); guardErr != nil {
			return runner.ExecutionResult{}, guardErr
		}
	}
	return runner.ExecutionResult{RuntimeID: result.RuntimeID, Output: result.Output}, nil
}

func (e *nexusPackageExecutor) Review(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.ReviewVerdict, error) {
	st, err := e.n.OpenProject()
	if err != nil {
		return runner.ReviewVerdict{}, err
	}
	implementerID := pkg.AssignedAgent
	if implementerID == "" {
		return runner.ReviewVerdict{}, fmt.Errorf("cannot review package without implementer identity")
	}

	reviewer, err := e.ensureReviewerAgent(ctx, st, run, pkg, implementerID)
	if err != nil {
		return runner.ReviewVerdict{}, err
	}

	reviewPrompt := buildReviewPrompt(run, pkg)
	before, _ := workspaceFingerprint(ctx, pkg.Workspace)
	result, err := e.n.executeAgentPrompt(ctx, reviewer.ID, pkg.Workspace, reviewPrompt, agentPromptPolicy{Contract: run.Contract, Review: true})
	if err != nil {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer execution failed: %w", err)
	}
	after, _ := workspaceFingerprint(ctx, pkg.Workspace)
	if before != "" && after != "" && before != after {
		return runner.ReviewVerdict{}, fmt.Errorf("independent reviewer modified the implementation workspace; review rejected")
	}

	verdict, err := parseReviewVerdict(result.Output)
	if err != nil {
		return runner.ReviewVerdict{}, err
	}
	verdict.ReviewerAgentID = reviewer.ID
	verdict.ReviewedAt = time.Now().UTC()
	return verdict, nil
}

func (e *nexusPackageExecutor) ensureReviewerAgent(ctx context.Context, st *store.Store, run *runner.MissionRun, pkg *runner.PackageRun, implementerID string) (store.Agent, error) {
	agents, err := st.ListAgents(run.ProjectID)
	if err != nil {
		return store.Agent{}, err
	}
	var reviewer store.Agent
	for _, candidate := range agents {
		if candidate.ID != implementerID && candidate.Role == "reviewer" && candidate.Name == "Nexus Independent Reviewer" {
			reviewer = candidate
			break
		}
	}
	if reviewer.ID == "" {
		reviewer, err = st.CreateAgent(store.Agent{ProjectID: run.ProjectID, Name: "Nexus Independent Reviewer", Role: "reviewer"})
		if err != nil {
			return store.Agent{}, err
		}
	}

	req := TaskRequirements{TaskKind: "review", Role: "reviewer", RequiredCapabilities: []string{"headless", "submit_prompt"}}
	accounts, err := e.n.ListResources()
	if err != nil {
		return store.Agent{}, err
	}
	reviewAccounts := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if supportsSafeHeadlessReview(account.Provider) {
			reviewAccounts = append(reviewAccounts, account)
		}
	}
	accounts = reviewAccounts

	// Independence means a distinct Agent identity and, whenever possible, a
	// distinct provider/profile from the implementer. Do not accidentally reward
	// same-provider affinity through the general scheduler for review work.
	implementerProvider := ""
	implementerProfile := ""
	if impl, getErr := st.GetAgent(implementerID, run.ProjectID); getErr == nil {
		if cfg, cfgErr := currentAgentConfig(st, impl); cfgErr == nil {
			implementerProvider = cfg.Provider
			implementerProfile = cfg.Profile
		}
	}
	if implementerProvider != "" {
		different := make([]ProviderAccount, 0, len(accounts))
		for _, account := range accounts {
			if account.Provider != implementerProvider || account.Profile != implementerProfile {
				different = append(different, account)
			}
		}
		if len(different) > 0 {
			accounts = different
		}
	}
	recommendation := RecommendResources(accounts, req, PolicyBalanced)
	if recommendation.Recommended == nil {
		return store.Agent{}, fmt.Errorf("no eligible reviewer provider/profile")
	}
	selected := recommendation.Recommended.Account
	current, _ := currentAgentConfig(st, reviewer)
	current.Provider = selected.Provider
	current.Profile = selected.Profile
	// Review is executed against the implementer's concrete worktree and must
	// not create a second unrelated worktree.
	current.Workspace = pkg.Workspace
	current.Isolation = "project"
	if _, err := e.n.SafeApply(ctx, reviewer.ID, current); err != nil {
		return store.Agent{}, fmt.Errorf("configure independent reviewer: %w", err)
	}
	return reviewer, nil
}

func supportsSafeHeadlessReview(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude", "gemini", "cursor":
		return true
	default:
		return false
	}
}

func missionTaskRequirements(pkg *runner.PackageRun) TaskRequirements {
	req := TaskRequirements{
		TaskKind:             "coding",
		Role:                 defaultRole(pkg.Role, "implementer"),
		RequiredCapabilities: []string{"headless", "submit_prompt"},
	}
	if strings.TrimSpace(pkg.TaskRequirements) != "" {
		var explicit TaskRequirements
		if json.Unmarshal([]byte(pkg.TaskRequirements), &explicit) == nil {
			if explicit.TaskKind != "" {
				req.TaskKind = explicit.TaskKind
			}
			if explicit.Role != "" {
				req.Role = explicit.Role
			}
			req.EstimatedTokens = explicit.EstimatedTokens
			req.PreferProvider = explicit.PreferProvider
			req.AgentPreference = explicit.AgentPreference
			req.ProjectPolicy = explicit.ProjectPolicy
			req.RequiredCapabilities = mergeRequiredCapabilities(req.RequiredCapabilities, explicit.RequiredCapabilities)
		}
	}
	return req
}

func mergeRequiredCapabilities(base, extra []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range append(append([]string(nil), base...), extra...) {
		key := normalizeCapabilityName(item)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func selectCurrentResource(accounts []ProviderAccount, cfg AgentConfig, req TaskRequirements) (ProviderAccount, bool) {
	if cfg.Provider == "" || cfg.Profile == "" {
		return ProviderAccount{}, false
	}
	for _, account := range accounts {
		if account.Provider != cfg.Provider || account.Profile != cfg.Profile {
			continue
		}
		candidate := evaluateCandidate(account, req, PolicyBalanced)
		return account, candidate.Eligible
	}
	return ProviderAccount{}, false
}

func currentAgentConfig(st *store.Store, agent store.Agent) (AgentConfig, error) {
	if agent.CurrentRevisionID == "" {
		return AgentConfig{}, nil
	}
	rev, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		return AgentConfig{}, err
	}
	return ParseAgentConfig(rev.Config)
}

func defaultRole(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func buildReviewPrompt(run *runner.MissionRun, pkg *runner.PackageRun) string {
	var evidence []string
	for _, v := range pkg.Verifications {
		evidence = append(evidence, fmt.Sprintf("- %s: passed=%t exit=%d\n%s", v.Command, v.Passed, v.ExitCode, v.OutputSnippet))
	}
	return fmt.Sprintf(`You are the independent code reviewer for an autonomous software-delivery mission.
Do NOT modify files, run destructive git commands, or approve based on assumptions.
Inspect the current workspace and implementation for the package below.

Package: %s
Goal: %s
Acceptance criteria:
- %s

Verification evidence:
%s

Return ONLY one JSON object with this exact schema:
{"approved":true|false,"findings":["..."],"remediation_tips":["..."]}
Approve only when the implementation genuinely satisfies the goal and acceptance criteria and you found no blocking correctness, security, test, or integration issue.`, pkg.Title, pkg.Goal, strings.Join(pkg.AcceptanceCriteria, "\n- "), strings.Join(evidence, "\n"))
}

func parseReviewVerdict(output string) (runner.ReviewVerdict, error) {
	text := strings.TrimSpace(output)
	if text == "" {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer returned no output")
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer did not return structured JSON evidence")
	}
	var payload struct {
		Approved        bool     `json:"approved"`
		Findings        []string `json:"findings"`
		RemediationTips []string `json:"remediation_tips"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return runner.ReviewVerdict{}, fmt.Errorf("invalid reviewer JSON: %w", err)
	}
	if payload.Approved && len(payload.Findings) == 0 {
		payload.Findings = []string{"Independent reviewer reported no blocking findings."}
	}
	return runner.ReviewVerdict{Approved: payload.Approved, Findings: payload.Findings, RemediationTips: payload.RemediationTips}, nil
}
