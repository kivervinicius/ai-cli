//go:build linux

package runtime

// linuxCredentialIsolation implements CredentialIsolation using private D-Bus and gnome-keyring-daemon.
type linuxCredentialIsolation struct{}

func newPlatformCredentialIsolation() CredentialIsolation { return &linuxCredentialIsolation{} }

func (l *linuxCredentialIsolation) Platform() string { return "linux" }

func (l *linuxCredentialIsolation) Capability() CredentialCapability {
	if _, err := LookPath("dbus-run-session"); err != nil {
		return CredentialCapability{Status: CredentialUnsupported, Mechanism: "DBus Secret Service", Reason: "dbus-run-session is unavailable"}
	}
	if _, err := LookPath("gnome-keyring-daemon"); err != nil {
		return CredentialCapability{Status: CredentialDegraded, Mechanism: "DBus Secret Service", Reason: "gnome-keyring-daemon is unavailable; provider isolation is incomplete"}
	}
	return CredentialCapability{Status: CredentialSupported, Mechanism: "private DBus session with Secret Service"}
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
	return dbus, append(wrapped, args...)
}
