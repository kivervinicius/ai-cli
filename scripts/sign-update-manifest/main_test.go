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
