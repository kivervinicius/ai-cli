package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/contextsnapshot"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

const missionEvidenceStreamName = "mission-validation"

// recordMissionValidationEvidence binds a concrete verification result to the
// repository identity observed at the time of the check. It deliberately
// records failed checks too: a failed claim is still useful recovery evidence,
// while callers must never promote it to VERIFIED.
func (n *Nexus) recordMissionValidationEvidence(ctx context.Context, run *runner.MissionRun, scenario, command string, results []runner.VerificationResult, provider, profile, model string, attempt int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("mission run is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	project, err := st.GetProject(run.ProjectID)
	if err != nil {
		return fmt.Errorf("load evidence project: %w", err)
	}
	identity, err := contextsnapshot.InspectCodeIdentity(project.CanonicalPath)
	if err != nil {
		return fmt.Errorf("inspect evidence identity: %w", err)
	}

	outcome, confidence, exitCode := store.EvidencePass, store.EvidenceVerified, (*int)(nil)
	passed := true
	for _, result := range results {
		if !result.Passed {
			passed = false
			value := result.ExitCode
			exitCode = &value
			break
		}
	}
	if !passed || len(results) == 0 || identity.HeadSHA == "" {
		outcome = store.EvidenceNotVerified
		confidence = store.EvidenceObserved
	}
	environment, _ := json.Marshal(map[string]string{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH})
	evidence, err := json.Marshal(map[string]any{"run_id": run.ID, "plan_id": run.PlanID, "plan_revision": run.PlanRevision, "results": results})
	if err != nil {
		return fmt.Errorf("encode validation evidence: %w", err)
	}

	stream, err := st.GetValidationEvidenceStreamByName(run.ProjectID, missionEvidenceStreamName)
	if err != nil {
		if !strings.Contains(err.Error(), store.ErrValidationEvidenceNotFound.Error()) {
			return err
		}
		stream, err = st.CreateValidationEvidenceStream(run.ProjectID, missionEvidenceStreamName)
		if err != nil {
			// Another worker may have won stream creation. Re-read before failing.
			stream, err = st.GetValidationEvidenceStreamByName(run.ProjectID, missionEvidenceStreamName)
			if err != nil {
				return err
			}
		}
	}
	entry := store.ValidationEvidenceEntry{
		ID: stableEvidenceID(run.ID, scenario, attempt, len(results)), GitSHA: identity.HeadSHA,
		IdentityDigest: identity.IdentityDigest, RepositoryState: string(identity.State), EnvironmentJSON: string(environment),
		Provider: provider, Profile: profile, Model: model, Scenario: scenario, CommandDisplay: command, Outcome: outcome, Confidence: confidence,
		ExitCode: exitCode, EvidenceJSON: string(evidence), CreatedAt: time.Now().UTC(),
	}
	if _, err := st.AppendValidationEvidence(store.ValidationEvidenceAppendRequest{StreamID: stream.ID, Entry: entry}); err != nil {
		return err
	}
	if err := st.VerifyValidationEvidenceChain(stream.ID); err != nil {
		return err
	}
	return nil
}

func stableEvidenceID(runID, scenario string, iteration, resultCount int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d", runID, scenario, iteration, resultCount)))
	return "mission_vee_" + hex.EncodeToString(sum[:])
}

func (e *nexusPackageExecutor) RecordValidationEvidence(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, results []runner.VerificationResult) error {
	if pkg == nil {
		return fmt.Errorf("package is required")
	}
	return e.n.recordMissionValidationEvidence(ctx, run, "mission/package/"+pkg.PackageID, strings.Join(pkg.VerificationRequirements, " && "), results, pkg.Provider, pkg.Profile, selectedModelFromRoutingDecision(pkg.RoutingDecisionJSON), pkg.Attempt)
}

func (e *nexusPackageExecutor) RecordGlobalValidationEvidence(ctx context.Context, run *runner.MissionRun, results []runner.VerificationResult) error {
	return e.n.recordMissionValidationEvidence(ctx, run, "mission/global-definition", strings.Join(run.Contract.GlobalVerificationCommands, " && "), results, "", "", "", run.TotalIterations)
}

func selectedModelFromRoutingDecision(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var decision struct {
		SelectedModel string `json:"selected_model"`
	}
	if json.Unmarshal([]byte(raw), &decision) != nil {
		return ""
	}
	return strings.TrimSpace(decision.SelectedModel)
}
