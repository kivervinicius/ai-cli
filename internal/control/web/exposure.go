package web

import "time"

// ExposureMode describes the effective reachability of the Control API.
// Bind address alone is insufficient: a loopback socket behind Cloudflare
// Tunnel is still PUBLIC_REMOTE.
type ExposureMode string

const (
	ExposureLoopback       ExposureMode = "LOOPBACK"
	ExposurePrivateNetwork ExposureMode = "PRIVATE_NETWORK"
	ExposurePublicRemote   ExposureMode = "PUBLIC_REMOTE"
)

// remoteBootstrapTTL bounds how long a printed bootstrap URL remains valid
// once the Core is effectively exposed beyond the local machine.
const remoteBootstrapTTL = 15 * time.Minute

// EffectiveExposure returns the security policy domain for auth decisions.
func (a *AuthManager) EffectiveExposure() ExposureMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.effectiveExposureLocked()
}

func (a *AuthManager) effectiveExposureLocked() ExposureMode {
	if a.tunnelActive {
		return ExposurePublicRemote
	}
	if a.isLoopbackListen() {
		return ExposureLoopback
	}
	return ExposurePrivateNetwork
}

// RequiresSecureCookie is true whenever cookies may traverse a non-local path.
func (a *AuthManager) RequiresSecureCookie() bool {
	return a.EffectiveExposure() != ExposureLoopback
}

// BootstrapReusable is true only for pure loopback (no tunnel).
func (a *AuthManager) BootstrapReusable() bool {
	return a.EffectiveExposure() == ExposureLoopback
}

func (a *AuthManager) now() time.Time {
	if a.clock != nil {
		return a.clock()
	}
	return time.Now()
}
