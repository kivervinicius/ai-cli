package provider

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/mock"
)

func TestProviderRegistry(t *testing.T) {
	reg := NewRegistry()

	fake1 := &mock.FakeProvider{
		ProviderID:   model.ProviderCodex,
		ProviderName: "Codex",
		DetectResult: model.DetectionResult{Installed: true, Version: "1.0.0"},
	}
	fake2 := &mock.FakeProvider{
		ProviderID:   model.ProviderClaude,
		ProviderName: "Claude Code",
		DetectResult: model.DetectionResult{Installed: false},
	}

	if err := reg.Register(fake1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(fake2); err != nil {
		t.Fatal(err)
	}

	// Duplicate register should fail
	if err := reg.Register(fake1); err == nil {
		t.Fatal("expected error registering duplicate provider")
	}

	p, found := reg.Get("codex")
	if !found || p.Name() != "Codex" {
		t.Fatalf("expected to get Codex provider, got %+v", p)
	}

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(list))
	}

	detections := reg.DetectAll(context.Background())
	if !detections["codex"].Installed || detections["claude"].Installed {
		t.Fatalf("unexpected detection results: %+v", detections)
	}
}
