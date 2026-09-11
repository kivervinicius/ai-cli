package tui

import (
	"strings"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestControlModelUsesInjectedControlRegistries(t *testing.T) {
	runtimeRegistry := registry.NewRegistry("")
	driverRegistry := driver.NewRegistry()
	model := NewControlModelWithDependencies(runtimeRegistry, driverRegistry)

	model.runtimes = []registry.RuntimeSession{{
		RuntimeID:  "runtime-injected",
		ProviderID: "codex",
		ProfileID:  "work",
	}}
	model.selectedIndex = 0

	if model.runtimeReg != runtimeRegistry || model.drivers != driverRegistry {
		t.Fatal("control model did not retain injected registries")
	}
	if strings.Contains(model.View(), "[h] Handoff") {
		t.Fatal("TUI exposed handoff capability from a global driver registry")
	}
}
