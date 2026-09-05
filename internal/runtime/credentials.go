package runtime

import (
	stdruntime "runtime"
)

// CredentialIsolation defines the interface for running commands with isolated platform credentials/keyrings.
type CredentialIsolation interface {
	// WrapCommand wraps a binary and arguments to isolate credential stores (e.g. secret service / keychain).
	WrapCommand(bin string, args []string) (string, []string)
	// Platform returns the operating system platform name this isolator supports.
	Platform() string
}

// DefaultCredentialIsolator returns the platform-appropriate CredentialIsolation implementation.
func DefaultCredentialIsolator() CredentialIsolation {
	switch stdruntime.GOOS {
	case "linux":
		return &linuxCredentialIsolation{}
	case "darwin":
		return &darwinCredentialIsolation{}
	case "windows":
		return &windowsCredentialIsolation{}
	default:
		return &noopCredentialIsolation{platform: stdruntime.GOOS}
	}
}

// linuxCredentialIsolation implements CredentialIsolation using private D-Bus and gnome-keyring-daemon.
type linuxCredentialIsolation struct{}

func (l *linuxCredentialIsolation) Platform() string {
	return "linux"
}

func (l *linuxCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	dbus, err := LookPath("dbus-run-session")
	if err != nil {
		return bin, args
	}
	if alreadyIsolatedSecretServiceArgv(args) {
		return dbus, args
	}
	keyring, err := LookPath("gnome-keyring-daemon")
	if err != nil {
		return dbus, append([]string{"--", bin}, args...)
	}
	wrapped := []string{"--", "/bin/sh", "-c", isolatedSecretServiceScript, "nexus-agy-keyring", keyring, bin}
	wrapped = append(wrapped, args...)
	return dbus, wrapped
}

// darwinCredentialIsolation handles macOS credential isolation.
type darwinCredentialIsolation struct{}

func (d *darwinCredentialIsolation) Platform() string {
	return "darwin"
}

func (d *darwinCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}

// windowsCredentialIsolation handles Windows credential isolation.
type windowsCredentialIsolation struct{}

func (w *windowsCredentialIsolation) Platform() string {
	return "windows"
}

func (w *windowsCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}

// noopCredentialIsolation is a fallback isolator for unknown platforms.
type noopCredentialIsolation struct {
	platform string
}

func (n *noopCredentialIsolation) Platform() string {
	return n.platform
}

func (n *noopCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}
