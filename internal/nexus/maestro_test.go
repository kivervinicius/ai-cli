package nexus

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMaestroUnavailableNeverFabricatesAdvice(t *testing.T) {
	c := &MaestroClient{status: MaestroStatus{Available: false, Mode: MaestroOff}, maestroBin: ""}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1"}, "ship product")
	if err == nil {
		t.Fatal("expected degraded error")
	}
	if !resp.Degraded || len(resp.Required) != 0 || len(resp.Recommended) != 0 || len(resp.Optional) != 0 {
		t.Fatalf("degraded Maestro must not fabricate recommendations: %+v", resp)
	}
}

func TestMaestroAdviseFailureNeverFallsBackToSyntheticRecommendations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	bin := filepath.Join(t.TempDir(), "maestro")
	script := "#!/bin/sh\nif [ \"$1\" = \"version\" ] || [ \"$1\" = \"--version\" ]; then echo 1.2.3; exit 0; fi\nexit 2\n"
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	c := &MaestroClient{status: MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.2.3"}}, maestroBin: bin}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1"}, "ship product")
	if err == nil {
		t.Fatal("expected advise contract failure")
	}
	if !resp.Degraded || len(resp.Required)+len(resp.Recommended)+len(resp.Optional) != 0 {
		t.Fatalf("expected empty degraded response, got %+v", resp)
	}
}

func TestFindOrquestradorDirUsesPortableDataDirProfiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	data := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	expected := filepath.Join(data, "profiles", "claude", "work", "home", ".orquestrador")
	if err := os.MkdirAll(expected, 0700); err != nil {
		t.Fatal(err)
	}
	if got := findOrquestradorDir(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
