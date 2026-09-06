//go:build darwin

package runtime

type darwinCredentialIsolation struct{}

func newPlatformCredentialIsolation() CredentialIsolation { return &darwinCredentialIsolation{} }
func (d *darwinCredentialIsolation) Platform() string     { return "darwin" }
func (d *darwinCredentialIsolation) Capability() CredentialCapability {
	return CredentialCapability{Status: CredentialUnsupported, Mechanism: "macOS Keychain", Reason: "native Keychain isolation is not implemented"}
}
func (d *darwinCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}
