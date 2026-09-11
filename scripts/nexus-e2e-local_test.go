package main

import (
	"errors"
	"strings"
	"testing"
)

func TestFindBootstrapRequiresTokenAndRedactsNothingInReturnedValue(t *testing.T) {
	bootstrap := findBootstrap("URL: http://127.0.0.1:13000\nBootstrap: http://127.0.0.1:13000/?token=abc123\n")
	if bootstrap == "" || !strings.Contains(bootstrap, "token=abc123") {
		t.Fatalf("bootstrap URL was not detected: %q", bootstrap)
	}
	if got := findBootstrap("Bootstrap: http://127.0.0.1:13000/\n"); got != "" {
		t.Fatalf("accepted tokenless bootstrap URL: %q", got)
	}
}

func TestParseBootstrapURLAcceptsPersistedFragmentToken(t *testing.T) {
	got := parseBootstrapURL("http://127.0.0.1:13000/#nexus_bootstrap=abc123")
	if got == "" || !strings.Contains(got, "#nexus_bootstrap=abc123") {
		t.Fatalf("persisted bootstrap URL was not accepted: %q", got)
	}
	if got := parseBootstrapURL("http://127.0.0.1:13000/"); got != "" {
		t.Fatalf("accepted tokenless URL: %q", got)
	}
}

func TestNotAuthenticatedIsMachineClassifiable(t *testing.T) {
	err := errors.Join(errNotAuthenticated, errors.New("provider unavailable"))
	if !errors.Is(err, errNotAuthenticated) {
		t.Fatal("authentication limitation is not classifiable")
	}
}

func TestProviderMarkerRequiresLiteralMarkerAndSanitizesANSI(t *testing.T) {
	if providerMarkerSeen("banner only\nNEXUS_E2E_O") {
		t.Fatal("accepted partial provider marker")
	}
	if !providerMarkerSeen("banner\n\x1b[32mNEXUS_E2E_OK\x1b[0m\n") {
		t.Fatal("did not accept literal marker with ANSI repaint")
	}
	if providerMarkerSeen("Reply with exactly NEXUS_E2E_OK and nothing else.") {
		t.Fatal("accepted marker echoed inside the prompt")
	}
	if providerMarkerSeen("NEXUS_E2E_NOT_OK") {
		t.Fatal("accepted non-marker output")
	}
	if strings.Contains(sanitizeTranscript("TOKEN=secret-value"), "secret-value") {
		t.Fatal("transcript secret was not redacted")
	}
}
