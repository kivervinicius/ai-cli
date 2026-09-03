package runtime

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestWrapWithIsolatedSecretServiceExportsPrivateKeyringDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private D-Bus wrapper is Unix-only")
	}
	_, args := WrapWithIsolatedSecretService("agy", nil)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, `export GNOME_KEYRING_CONTROL="$TMPDIR"`) {
		t.Fatalf("wrapper must export its private keyring directory: %s", joined)
	}
}

func TestIsolatedSecretServiceScriptIsValidPOSIX(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private D-Bus wrapper is Unix-only")
	}
	cmd := exec.Command("/bin/sh", "-n", "-c", isolatedSecretServiceScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper script is not valid POSIX sh: %v\n%s", err, out)
	}
}

func TestIsolatedSecretServiceScriptExecsProviderArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private D-Bus wrapper is Unix-only")
	}
	cmd := exec.Command("/bin/sh", "-c", isolatedSecretServiceScript, "nexus-agy-keyring", "/bin/true", "/bin/echo", "hello-from-wrapper")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper failed to exec provider: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "hello-from-wrapper" {
		t.Fatalf("expected provider output, got %q", got)
	}
}

func TestWrapDoesNotNestExistingDBusWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private D-Bus wrapper is Unix-only")
	}
	if _, err := LookPath("dbus-run-session"); err != nil {
		t.Skip("dbus-run-session not installed")
	}
	_, once := WrapWithIsolatedSecretService("/usr/bin/agy", []string{"-p", "hello"})
	_, twice := WrapWithIsolatedSecretService("/usr/bin/agy", once)
	shCount := 0
	for _, arg := range twice {
		if arg == "/bin/sh" {
			shCount++
		}
	}
	if shCount > 1 {
		t.Fatalf("nested wrap passed /bin/sh to the provider: %v", twice)
	}
	if len(twice) != len(once) {
		t.Fatalf("second wrap changed argv\n first=%v\nsecond=%v", once, twice)
	}
}

func TestNormalizeInteractiveEnvReplacesDumbTerminal(t *testing.T) {
	env := []string{"PATH=/bin", "TERM=dumb", "LANG=pt_BR.UTF-8"}
	normalized := NormalizeInteractiveEnv(env)

	if normalized[1] != "TERM=xterm-256color" {
		t.Fatalf("expected interactive terminal type, got %q", normalized[1])
	}
	if env[1] != "TERM=dumb" {
		t.Fatalf("normalization must not mutate caller environment, got %q", env[1])
	}
}

func TestNormalizeInteractiveEnvPreservesNormalTerminal(t *testing.T) {
	env := []string{"TERM=screen-256color"}
	normalized := NormalizeInteractiveEnv(env)

	if len(normalized) != 1 || normalized[0] != env[0] {
		t.Fatalf("normal terminal environment changed: %+v", normalized)
	}
}
