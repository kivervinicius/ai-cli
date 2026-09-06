//go:build windows

package runtime

type windowsCredentialIsolation struct{}

func newPlatformCredentialIsolation() CredentialIsolation { return &windowsCredentialIsolation{} }
func (w *windowsCredentialIsolation) Platform() string    { return "windows" }
func (w *windowsCredentialIsolation) Capability() CredentialCapability {
	return CredentialCapability{Status: CredentialUnsupported, Mechanism: "Windows Credential Manager", Reason: "native Credential Manager isolation is not implemented"}
}
func (w *windowsCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}
