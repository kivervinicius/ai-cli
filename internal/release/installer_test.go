package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerScriptSyntax(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash script syntax check requires unix shell")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	scripts := []string{"install.sh", "uninstall.sh", "scripts/package-linux-beta.sh"}
	for _, s := range scripts {
		scriptPath := filepath.Join(root, s)
		if _, err := os.Stat(scriptPath); err != nil {
			t.Errorf("script %s not found: %v", s, err)
			continue
		}
		cmd := exec.Command("bash", "-n", scriptPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("bash syntax error in %s: %v\nOutput:\n%s", s, err, string(output))
		}
	}
}

func TestInstallerArchiveNamingMatchesGoReleaser(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	goreleaserBytes, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("failed to read .goreleaser.yaml: %v", err)
	}
	goreleaserContent := string(goreleaserBytes)

	installShBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	installSh := string(installShBytes)

	installPs1Bytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatalf("failed to read install.ps1: %v", err)
	}
	installPs1 := string(installPs1Bytes)

	// Verify archive template and formats
	expectedPatterns := []struct {
		platform string
		archive  string
	}{
		{"Linux x86_64", "nexus_Linux_x86_64.tar.gz"},
		{"Linux arm64", "nexus_Linux_arm64.tar.gz"},
		{"Darwin x86_64", "nexus_Darwin_x86_64.tar.gz"},
		{"Darwin arm64", "nexus_Darwin_arm64.tar.gz"},
		{"Windows x86_64", "nexus_Windows_x86_64.zip"},
		{"Windows arm64", "nexus_Windows_arm64.zip"},
	}

	for _, p := range expectedPatterns {
		// Verify install.sh constructs nexus_${OS_NAME}_${ARCH_NAME}.tar.gz
		if !strings.Contains(installSh, `ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"`) {
			t.Errorf("install.sh missing archive format pattern for %s", p.platform)
		}
		// Verify install.ps1 constructs nexus_${OsName}_${Arch}.${ArchiveExt}
		if !strings.Contains(installPs1, `nexus_${OsName}_${Arch}.${ArchiveExt}`) {
			t.Errorf("install.ps1 missing archive format pattern for %s", p.platform)
		}
	}

	// Verify nfpms package section exists in goreleaser
	if !strings.Contains(goreleaserContent, "nfpms:") {
		t.Errorf(".goreleaser.yaml should define nfpms package section for Linux packages")
	}
}

func TestInstallerDoesNotSilentlyInstallMaestro(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	installShBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	installSh := string(installShBytes)

	installPs1Bytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatalf("failed to read install.ps1: %v", err)
	}
	installPs1 := string(installPs1Bytes)

	if !strings.Contains(installSh, "--with-maestro") {
		t.Error("install.sh must guard Maestro installation behind --with-maestro")
	}
	if !strings.Contains(installSh, `[ "$WITH_MAESTRO" = true ]`) {
		t.Error("install.sh must only execute Maestro installation when WITH_MAESTRO is true")
	}

	if !strings.Contains(installPs1, "WithMaestro") {
		t.Error("install.ps1 must declare WithMaestro parameter")
	}
	if !strings.Contains(installPs1, "if ($WithMaestro)") {
		t.Error("install.ps1 must only execute Maestro installation when $WithMaestro is true")
	}
}
