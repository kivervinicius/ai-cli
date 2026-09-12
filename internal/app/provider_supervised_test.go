package app

import (
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/nexus"
)

func TestProviderLaunchFlagsAreNexusOwned(t *testing.T) {
	args := []string{"--yolo", "--supervised", "--model", "test"}
	if !hasProviderLaunchFlag(args, "--supervised") {
		t.Fatal("expected supervised flag to be recognized")
	}
	got := removeProviderLaunchFlag(args, "--supervised")
	if len(got) != 3 || got[0] != "--yolo" || got[1] != "--model" || got[2] != "test" {
		t.Fatalf("provider args = %#v", got)
	}
	if hasProviderLaunchFlag(got, "--supervised") {
		t.Fatal("Nexus-owned flag leaked to provider args")
	}
}

func TestProfileInCandidates(t *testing.T) {
	candidates := []model.Profile{{Name: "agy1"}, {Name: "agy2"}}
	if !profileInCandidates(candidates, "agy2") {
		t.Fatal("expected configured profile")
	}
	if profileInCandidates(candidates, "codex1") {
		t.Fatal("unexpected profile match")
	}
}

func TestInteractiveDelegationModeUsesSessionAndEnvironmentPolicy(t *testing.T) {
	session := registry.RuntimeSession{Labels: map[string]string{
		"nexus.delegation_mode": string(nexus.DelegationAsk),
	}}
	if got := interactiveDelegationMode(session); got != nexus.DelegationAsk {
		t.Fatalf("session delegation mode = %q, want ASK", got)
	}

	t.Setenv("NEXUS_DELEGATION_MODE", string(nexus.DelegationOff))
	if got := interactiveDelegationMode(session); got != nexus.DelegationOff {
		t.Fatalf("environment delegation mode = %q, want OFF", got)
	}

	t.Setenv("NEXUS_DELEGATION_MODE", "unsupported")
	if got := interactiveDelegationMode(session); got != nexus.DelegationAuto {
		t.Fatalf("unsupported delegation mode = %q, want AUTO", got)
	}
}
