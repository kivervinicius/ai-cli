//go:build !linux && !darwin && !windows

package runtime

import stdruntime "runtime"

type noopCredentialIsolation struct{}

func newPlatformCredentialIsolation() CredentialIsolation { return &noopCredentialIsolation{} }
func (n *noopCredentialIsolation) Platform() string       { return stdruntime.GOOS }
func (n *noopCredentialIsolation) Capability() CredentialCapability {
	return CredentialCapability{Status: CredentialUnsupported, Mechanism: "none", Reason: "platform has no credential isolation implementation"}
}
func (n *noopCredentialIsolation) WrapCommand(bin string, args []string) (string, []string) {
	return bin, args
}
