package originpolicy

import "testing"

func TestValidateRequiresExactRequestTarget(t *testing.T) {
	cases := []struct {
		host, origin string
		ok           bool
	}{
		{"127.0.0.1:9090", "http://127.0.0.1:9090", true},
		{"localhost:9090", "http://localhost:9090", true},
		{"192.168.1.10:9090", "https://192.168.1.10:9090", true},
		{"192.168.1.10:9090", "https://192.168.1.11:9090", false},
		{"127.0.0.1:9090", "http://127.0.0.1:9091", false},
		{"127.0.0.1:9090", "http://localhost:9090", false},
		{"localhost.evil.com:9090", "http://localhost.evil.com:9090", false},
		{"8.8.8.8:9090", "http://8.8.8.8:9090", false},
	}
	for _, tc := range cases {
		if got := Validate(tc.host, tc.origin); got != tc.ok {
			t.Fatalf("host=%q origin=%q got=%v want=%v", tc.host, tc.origin, got, tc.ok)
		}
	}
}
