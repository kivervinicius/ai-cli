package doctor

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	nexusruntime "github.com/kivervinicius/ai-cli/internal/runtime"
)

func TestBuildReportIsReadOnlyAndContainsStableChecks(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	before, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	report := BuildReport("test", map[string]model.DetectionResult{"codex": {Installed: false, Error: "missing"}}, nexusruntime.CredentialCapability{Status: nexusruntime.CredentialUnsupported, Mechanism: "test"})
	after, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) || len(report.Checks) == 0 {
		t.Fatalf("doctor mutated state or returned no checks: before=%d after=%d", len(before), len(after))
	}
}

func TestWriteBundleAllowlistAndRedactionSurface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.zip")
	report := Report{Schema: "nexus.doctor/v1", Version: "test", Providers: map[string]model.DetectionResult{"codex": {Installed: true, BinaryPath: "/tmp/codex"}}}
	if err := report.WriteBundle(path); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 2 {
		t.Fatalf("unexpected bundle entries: %d", len(archive.File))
	}
	for _, entry := range archive.File {
		if entry.Name != "report.json" && entry.Name != "MANIFEST.txt" {
			t.Fatalf("unexpected bundle entry %q", entry.Name)
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == "SECRET_CANARY" {
			t.Fatal("diagnostic bundle leaked secret canary")
		}
	}
}
