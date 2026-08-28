package web

import (
	"fmt"
	"net"
)

// cgnatRange is the CGNAT shared-address range 100.64.0.0/10 (RFC 6598).
// net.IP.IsPrivate() does NOT cover it, so it is handled explicitly.
var cgnatRange = net.IPNet{
	IP:   net.ParseIP("100.64.0.0"),
	Mask: net.CIDRMask(10, 32),
}

// ValidateBind enforces the loopback-default binding policy.
//
//   - loopback addresses: always allowed
//   - private / CGNAT / link-local addresses: allowed only with an explicit
//     --remote opt-in (strong auth + Origin policy remain enforced)
//   - public addresses: always refused — no TLS/identity solution exists yet
func ValidateBind(host string, remote bool) error {
	if host == "" || host == "localhost" {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("refusing to bind to host %q: only IP addresses are accepted for --listen", host)
	}
	if ip.IsLoopback() {
		return nil
	}
	if isPrivateIP(ip) {
		if remote {
			return nil
		}
		return fmt.Errorf("refusing to bind to private address %s: pass --remote to explicitly allow non-loopback binding; prefer SSH port forwarding (ssh -N -L local:127.0.0.1:port user@machine) or a private VPN", host)
	}
	return fmt.Errorf("refusing to bind to public address %s: the Web Control Center must not be exposed on a public interface without an explicit TLS/identity solution; use an SSH tunnel or a private VPN instead", host)
}

// isPrivateIP reports whether ip is RFC1918, ULA, link-local, or CGNAT.
func isPrivateIP(ip net.IP) bool {
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return cgnatRange.Contains(ip)
}
