package doctor

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	nexusruntime "github.com/kivervinicius/ai-cli/internal/runtime"
)

type Status string

const (
	Pass    Status = "PASS"
	Warn    Status = "WARN"
	Fail    Status = "FAIL"
	Skipped Status = "SKIPPED"
)

type Check struct {
	ID          string `json:"id"`
	Status      Status `json:"status"`
	Summary     string `json:"summary"`
	Remediation string `json:"remediation,omitempty"`
}

type Report struct {
	Schema      string                            `json:"schema"`
	GeneratedAt time.Time                         `json:"generated_at"`
	Version     string                            `json:"version"`
	OS          string                            `json:"os"`
	Arch        string                            `json:"arch"`
	Checks      []Check                           `json:"checks"`
	Providers   map[string]model.DetectionResult  `json:"providers"`
	Credentials nexusruntime.CredentialCapability `json:"credentials"`
}

// BuildReport is read-only. It inspects existing state and never creates a
// project, registry row, profile, socket or temporary persistent directory.
func BuildReport(version string, detections map[string]model.DetectionResult, capability nexusruntime.CredentialCapability) Report {
	report := Report{
		Schema:      "nexus.doctor/v1",
		GeneratedAt: time.Now().UTC(),
		Version:     version,
		OS:          stdruntime.GOOS,
		Arch:        stdruntime.GOARCH,
		Providers:   detections,
		Credentials: capability,
	}
	report.Checks = append(report.Checks, checkDirectory("data_directory", config.DataDir), checkDirectory("config_directory", config.ConfigDir), checkDirectory("state_directory", config.StateDir))
	report.Checks = append(report.Checks, Check{ID: "credentials.capability", Status: capabilityStatus(capability.Status), Summary: string(capability.Status) + " — " + capability.Mechanism, Remediation: capability.Reason})
	providerIDs := make([]string, 0, len(detections))
	for id := range detections {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	for _, id := range providerIDs {
		detection := detections[id]
		status := Pass
		if !detection.Installed {
			status = Warn
		}
		report.Checks = append(report.Checks, Check{ID: "provider." + id, Status: status, Summary: providerSummary(detection)})
	}
	return report
}

func checkDirectory(id string, resolver func() (string, error)) Check {
	path, err := resolver()
	if err != nil || path == "" {
		return Check{ID: id, Status: Fail, Summary: "directory could not be resolved", Remediation: "inspect Nexus directory environment configuration"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return Check{ID: id, Status: Warn, Summary: "directory is not present: " + path, Remediation: "start Nexus once or create the directory with secure permissions"}
	}
	if !info.IsDir() {
		return Check{ID: id, Status: Fail, Summary: "path is not a directory: " + path}
	}
	return Check{ID: id, Status: Pass, Summary: "directory is available: " + path}
}

func capabilityStatus(status nexusruntime.CredentialCapabilityStatus) Status {
	switch status {
	case nexusruntime.CredentialSupported:
		return Pass

	case nexusruntime.CredentialDegraded, nexusruntime.CredentialUnsupported:
		return Warn
	default:
		return Fail
	}
}

func providerSummary(detection model.DetectionResult) string {
	if !detection.Installed {
		return "not installed: " + detection.Error
	}
	return fmt.Sprintf("installed (%s)", detection.Version)
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// WriteBundle writes only an allowlisted report. It intentionally does not
// archive environment variables, arguments, prompts, transcripts or provider
// logs.
func (r Report) WriteBundle(path string) error {
	data, err := r.JSON()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	entry, err := archive.Create("report.json")
	if err != nil {
		return err
	}
	if _, err := entry.Write(data); err != nil {
		return err
	}
	manifest, err := archive.Create("MANIFEST.txt")
	if err != nil {
		return err
	}
	if _, err := manifest.Write([]byte("report.json\n")); err != nil {
		return err
	}
	return archive.Close()
}
