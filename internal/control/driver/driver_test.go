package driver

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestDriverRegistryAndCapabilities(t *testing.T) {
	reg := DefaultRegistry()

	providers := []string{"codex", "agy", "claude", "opencode", "gemini"}
	for _, p := range providers {
		d, err := reg.Get(p)
		if err != nil {
			t.Fatalf("driver not found for %s: %v", p, err)
		}
		if d.ProviderID() != p {
			t.Errorf("expected provider ID %s, got %s", p, d.ProviderID())
		}

		caps := d.Capabilities(context.Background(), model.Profile{Name: "default"})
		if !caps.Process || !caps.Terminal || !caps.SlashControl {
			t.Errorf("driver %s missing core capabilities: %+v", p, caps)
		}
	}

	// Unknown provider check
	if _, err := reg.Get("non-existent"); err == nil {
		t.Errorf("expected error for non-existent driver")
	}
}
