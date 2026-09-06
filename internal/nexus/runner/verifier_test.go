package runner

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestVerificationEnginePreservesQuotedShellArguments(t *testing.T) {
	cmd := `printf '%s' 'hello world'`
	if runtime.GOOS == "windows" {
		cmd = `echo hello world`
	}
	results := NewVerificationEngine().RunVerification(context.Background(), t.TempDir(), []string{cmd})
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("verification failed: %+v", results)
	}
	if !strings.Contains(results[0].OutputSnippet, "hello world") {
		t.Fatalf("quoted command was not preserved: %q", results[0].OutputSnippet)
	}
}

func TestDefaultAutonomyContractHasBoundedPackageTimeout(t *testing.T) {
	contract := DefaultAutonomyContract()
	if contract.PackageTimeoutSeconds < 60 {
		t.Fatalf("package timeout must be bounded and practical, got %d", contract.PackageTimeoutSeconds)
	}
	if contract.MaxTotalIterations < 50 {
		t.Fatalf("mission iteration budget too small for multi-package lifecycle: %d", contract.MaxTotalIterations)
	}
}
