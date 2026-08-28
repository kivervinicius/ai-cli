package web

import "testing"

func TestValidateBindLoopback(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "", "localhost"} {
		if err := ValidateBind(host, false); err != nil {
			t.Errorf("loopback host %q should be allowed without --remote, got: %v", host, err)
		}
	}
}

func TestValidateBindPrivateRequiresRemote(t *testing.T) {
	cases := []string{
		"192.168.1.10",
		"10.0.0.5",
		"172.16.0.3",
		"100.64.0.7",    // CGNAT
		"100.127.255.1", // CGNAT upper bound
		"fc00::1",       // ULA
		"169.254.10.1",  // link-local
	}
	for _, host := range cases {
		if err := ValidateBind(host, false); err == nil {
			t.Errorf("private host %q must be refused without --remote", host)
		}
		if err := ValidateBind(host, true); err != nil {
			t.Errorf("private host %q with --remote should be allowed, got: %v", host, err)
		}
	}
}

func TestValidateBindPublicAlwaysRefused(t *testing.T) {
	for _, host := range []string{"8.8.8.8", "1.1.1.1", "200.150.10.5", "2606:4700::1111"} {
		if err := ValidateBind(host, false); err == nil {
			t.Errorf("public host %q must be refused without --remote", host)
		}
		if err := ValidateBind(host, true); err == nil {
			t.Errorf("public host %q must be refused even with --remote", host)
		}
	}
}

func TestValidateBindHostnameRefused(t *testing.T) {
	if err := ValidateBind("example.com", true); err == nil {
		t.Error("hostname must be refused")
	}
}
