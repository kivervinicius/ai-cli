package main

import "testing"

func TestReleaseArtifactMetadataCanonicalizesGoReleaserNames(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		target string
	}{
		{"nexus_Linux_x86_64.tar.gz", "linux_amd64", "tar.gz"},
		{"nexus_Windows_arm64.zip", "windows_arm64", "zip"},
		{"nexus_0.5.0-beta.23_Darwin_amd64.deb", "darwin_amd64_deb", "deb"},
		{"nexus_0.5.0-beta.23_linux_arm64.rpm", "linux_arm64_rpm", "rpm"},
	}
	for _, test := range tests {
		key, target, ok := releaseArtifactMetadata(test.name)
		if !ok || key != test.key || target != test.target {
			t.Errorf("releaseArtifactMetadata(%q) = (%q, %q, %t), want (%q, %q, true)", test.name, key, target, ok, test.key, test.target)
		}
	}
}

func TestReleaseArtifactMetadataRejectsUnpublishableFiles(t *testing.T) {
	for _, name := range []string{"checksums.txt", "nexus_linux_386.tar.gz", "nexus_unknown_amd64.zip"} {
		if _, _, ok := releaseArtifactMetadata(name); ok {
			t.Errorf("releaseArtifactMetadata(%q) accepted an unsupported artifact", name)
		}
	}
}

func TestArtifactKeysMapGoReleaserAndDesktopNames(t *testing.T) {
	cases := map[string][]string{
		"nexus_Linux_x86_64.tar.gz":           {"linux_amd64"},
		"nexus_Darwin_arm64.tar.gz":           {"darwin_arm64"},
		"nexus_Windows_x86_64.zip":            {"windows_amd64"},
		"nexus-desktop_Linux_x86_64.tar.gz":   {"desktop_linux_amd64"},
		"nexus-desktop_Darwin_arm64.zip":      {"desktop_darwin_arm64"},
		"nexus-setup-Windows_x86_64.exe":      {"windows_amd64_nsis"},
		"nexus_0.5.0-beta.23_linux_amd64.deb": {"linux_amd64_deb"},
		"nexus_0.5.0-beta.23_linux_amd64.rpm": {"linux_amd64_rpm"},
	}
	for name, mustContain := range cases {
		keys := artifactKeys(name)
		for _, want := range mustContain {
			found := false
			for _, got := range keys {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s keys=%v missing %s", name, keys, want)
			}
		}
		if artifactTarget(name) == "" {
			t.Fatalf("%s missing target", name)
		}
	}
}
