package release

import "testing"

func TestNextVersion(t *testing.T) {
	tests := []struct {
		name, current, kind, want string
	}{
		{"patch", "0.4.6", "patch", "0.4.7"},
		{"minor", "0.4.6", "minor", "0.5.0"},
		{"major", "0.4.6", "major", "1.0.0"},
		{"new beta", "0.4.6", "beta", "0.5.0-beta.0"},
		{"next beta", "0.5.0-beta.0", "beta", "0.5.0-beta.1"},
		{"new rc", "0.5.0-beta.2", "rc", "0.5.0-rc.0"},
		{"next rc", "0.5.0-rc.0", "rc", "0.5.0-rc.1"},
		{"stable", "0.5.0-rc.1", "stable", "0.5.0"},
		{"hotfix", "0.4.6", "hotfix", "0.4.7-beta.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextVersion(tt.current, tt.kind)
			if err != nil || got != tt.want {
				t.Fatalf("NextVersion(%q, %q) = %q, %v; want %q", tt.current, tt.kind, got, err, tt.want)
			}
		})
	}
}

func TestValidateAndOptions(t *testing.T) {
	if err := Validate("1.2.3-beta.4+build.5"); err != nil {
		t.Fatalf("valid prerelease rejected: %v", err)
	}
	if err := Validate("1.2"); err == nil {
		t.Fatal("invalid SemVer accepted")
	}
	if err := Validate("v1.2.3"); err == nil {
		t.Fatal("version with v prefix accepted")
	}
	if got := Options("0.5.0-beta.2"); len(got) != 4 || got[0].Kind != "beta" || got[len(got)-1].Kind != "custom" {
		t.Fatalf("unexpected prerelease options: %+v", got)
	}
	if got := Options("0.5.0-rc.1"); len(got) != 3 || got[0].Kind == "beta" {
		t.Fatalf("RC should not offer beta downgrade: %+v", got)
	}
	if got := Options("0.5.0"); len(got) != 6 {
		t.Fatalf("unexpected stable options: %+v", got)
	}
}
