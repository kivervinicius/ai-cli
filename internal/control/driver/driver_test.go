package driver

import (
	"context"
	"reflect"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestDriverRegistryAndCapabilities(t *testing.T) {
	reg := DefaultRegistry()
	ctx := context.Background()

	providers := []string{"codex", "agy", "claude", "opencode", "gemini", "fake"}
	for _, p := range providers {
		d, err := reg.Get(p)
		if err != nil {
			t.Fatalf("driver not found for %s: %v", p, err)
		}
		if d.ProviderID() != p {
			t.Errorf("expected provider ID %s, got %s", p, d.ProviderID())
		}

		effCaps := d.EffectiveCaps(ctx, model.Profile{Name: "default"})
		if effCaps.Process.Status != CapabilitySupported || effCaps.Terminal.Status != CapabilitySupported {
			t.Errorf("driver %s expected Process and Terminal supported, got %+v", p, effCaps)
		}

		// Truthful test: Codex and OpenCode without server adapters should NOT claim StructuredEvents supported
		if p == "codex" || p == "opencode" {
			if effCaps.StructuredEvents.Status == CapabilitySupported {
				t.Errorf("driver %s must not claim StructuredEvents supported without real adapter", p)
			}
		}

		// Test CanResume with valid and empty session ID
		canResumeEmpty, reason := d.CanResume(ctx, model.Profile{Name: "default"}, "")
		if canResumeEmpty {
			t.Errorf("driver %s should reject empty session ID for resume", p)
		}
		if reason == "" {
			t.Errorf("driver %s should provide reason when resume is rejected", p)
		}

		canResumeValid, _ := d.CanResume(ctx, model.Profile{Name: "default"}, "sess-123")
		if !canResumeValid {
			t.Errorf("driver %s should allow valid session ID", p)
		}

		resumeArgs, err := d.BuildResumeArgs(ctx, model.Profile{Name: "default"}, "sess-123")
		if err != nil || len(resumeArgs) == 0 {
			t.Errorf("driver %s failed to build resume args: %v", p, err)
		}
	}

	// Unknown provider check
	if _, err := reg.Get("non-existent"); err == nil {
		t.Errorf("expected error for non-existent driver")
	}
}

func TestHeadlessKickoffArgsMatchProviderCLIContracts(t *testing.T) {
	ctx := context.Background()
	profile := model.Profile{Name: "default"}
	cases := []struct {
		name string
		d    ControlDriver
		want []string
	}{
		{name: "claude", d: NewClaudeDriver(), want: []string{"-p", "ship it"}},
		{name: "agy", d: NewAGYDriver(), want: []string{"-p", "ship it"}},
		{name: "gemini", d: NewGeminiDriver(), want: []string{"-p", "ship it"}},
		{name: "opencode", d: NewOpenCodeDriver(), want: []string{"run", "ship it"}},
		{name: "cursor", d: NewCursorDriver(), want: []string{"-p", "ship it", "--output-format", "text"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.BuildKickoffArgs(ctx, profile, "ship it")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("kickoff args mismatch: got %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestAutonomousArgsAreOwnedByDrivers(t *testing.T) {
	ctx := context.Background()
	profile := model.Profile{Name: "default"}
	tests := []struct {
		name    string
		driver  ControlDriver
		mode    AutonomousMode
		auto    bool
		want    []string
		wantErr bool
	}{
		{name: "claude coding", driver: NewClaudeDriver(), mode: AutonomousCoding, auto: true, want: []string{"-p", "ship it", "--dangerously-skip-permissions", "--output-format", "text"}},
		{name: "claude review", driver: NewClaudeDriver(), mode: AutonomousReview, want: []string{"-p", "ship it", "--permission-mode", "plan", "--output-format", "text"}},
		{name: "gemini coding", driver: NewGeminiDriver(), mode: AutonomousCoding, auto: true, want: []string{"-p", "ship it", "--approval-mode=yolo", "--output-format", "text"}},
		{name: "gemini review", driver: NewGeminiDriver(), mode: AutonomousReview, want: []string{"-p", "ship it", "--approval-mode=plan", "--output-format", "text"}},
		{name: "agy coding", driver: NewAGYDriver(), mode: AutonomousCoding, auto: true, want: []string{"-p", "ship it", "--dangerously-skip-permissions", "--sandbox", "--print-timeout", "60m"}},
		{name: "agy review unsupported", driver: NewAGYDriver(), mode: AutonomousReview, wantErr: true},
		{name: "opencode coding", driver: NewOpenCodeDriver(), mode: AutonomousCoding, auto: true, want: []string{"run", "--auto", "ship it"}},
		{name: "opencode review unsupported", driver: NewOpenCodeDriver(), mode: AutonomousReview, wantErr: true},
		{name: "cursor coding", driver: NewCursorDriver(), mode: AutonomousCoding, auto: true, want: []string{"-p", "ship it", "--output-format", "text"}},
		{name: "cursor review", driver: NewCursorDriver(), mode: AutonomousReview, want: []string{"-p", "ship it", "--output-format", "text", "--mode=ask"}},
		{name: "codex autonomous unsupported", driver: NewCodexDriver(), mode: AutonomousCoding, auto: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.driver.BuildAutonomousArgs(ctx, profile, "ship it", tc.mode, AutonomousPolicy{AllowToolAutoApproval: tc.auto})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got args %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("autonomous args mismatch: got %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestAutonomousCodingRequiresExplicitToolApproval(t *testing.T) {
	_, err := NewClaudeDriver().BuildAutonomousArgs(context.Background(), model.Profile{Name: "default"}, "ship it", AutonomousCoding, AutonomousPolicy{})
	if err == nil {
		t.Fatal("expected autonomous coding to require explicit tool auto approval")
	}
}
