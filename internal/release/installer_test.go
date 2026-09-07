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

func TestInstallersRequirePinnedArtifactsOrExplicitSourceBuild(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	sh, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	shText, psText := string(sh), string(ps)

	for _, forbidden := range []string{"releases/latest", "@latest", "git clone --depth 1 \"${GITHUB_URL}.git\""} {
		if strings.Contains(shText, forbidden) {
			t.Errorf("install.sh must not use mutable source/artifact reference %q", forbidden)
		}
	}
	for _, forbidden := range []string{"releases/latest", "@latest", "git clone --depth 1 \"$GithubUrl.git\""} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 must not use mutable source/artifact reference %q", forbidden)
		}
	}

	for _, required := range []string{"--version=", "--build-from-source", "checksums.txt", "sha256sum", "shasum -a 256"} {
		if !strings.Contains(shText, required) {
			t.Errorf("install.sh missing pinned/digest guard %q", required)
		}
	}
	for _, required := range []string{"-Version", "-BuildFromSource", "checksums.txt", "Get-FileHash"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 missing pinned/digest guard %q", required)
		}
	}
	for _, required := range []string{"Ensure-GoCompiler", "winget", "GoLang.Go", "go.dev/dl"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 must install or explain the Go source-build dependency: missing %q", required)
		}
	}
}

func TestPowerShellInstallerUsesNexusPathAndPreservesLegacyPath(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		`"Programs\IAPro Nexus"`,
		`"Programs\ai-cli"`,
		"leaving it untouched",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("install.ps1 must preserve legacy compatibility while using Nexus branding: missing %q", required)
		}
	}
}

func TestPublicLicenseMetadataMatchesLicenseFile(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(license), "MIT License") {
		t.Fatalf("expected the vigente LICENSE file to be MIT, got %q", strings.SplitN(string(license), "\n", 2)[0])
	}

	for _, name := range []string{"README.md", "README.en.md", "README.es.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(strings.ToLower(text), "apache") {
			t.Errorf("%s still advertises Apache despite the MIT LICENSE", name)
		}
		if !strings.Contains(strings.ToLower(text), "mit") {
			t.Errorf("%s does not advertise the vigente MIT license", name)
		}
	}
}
