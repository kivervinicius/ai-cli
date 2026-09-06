package runner

import "context"

// PackageExecutor bridges the deterministic runner to real Nexus Agents/providers.
// No method has a success fallback: missing execution/review capability is an error.
type PackageExecutor interface {
	Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error)
	Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error)
	Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error)
	Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error)
}
