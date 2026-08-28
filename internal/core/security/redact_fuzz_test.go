package security

import (
	"strings"
	"testing"
)

func FuzzRedact(f *testing.F) {
	seeds := []string{
		"OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890abcdef1234567890",
		"ANTHROPIC_API_KEY=sk-ant-api03-abcdef1234567890abcdef1234567890-test",
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0In0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0...\n-----END RSA PRIVATE KEY-----",
		"postgres://dbuser:s3cr3tP@ss!@localhost:5432/mydb",
		"GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
		"Cookie: sessionid=abc123; csrftoken=xyz789",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		out := Redact(input)
		// Redaction must never panic or collapse non-empty input to empty.
		if input != "" && out == "" {
			t.Fatalf("Redact returned empty for non-empty input")
		}
		// A redaction marker followed by a marker close must never contain a raw secret fragment.
		for _, marker := range []string{"[REDACTED_OPENAI_KEY]", "[REDACTED_ANTHROPIC_KEY]", "[REDACTED_JWT_TOKEN]", "[REDACTED_PRIVATE_KEY]"} {
			if strings.Contains(out, marker+"sk-") || strings.Contains(out, marker+"eyJ") {
				t.Fatalf("Redact output leaks secret adjacent to marker: %q", out)
			}
		}
	})
}
