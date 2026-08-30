package originpolicy

import (
	"net"
	"net/url"
	"strings"
)

// Validate accepts browser origins only when they target the exact HTTP Host
// of the request and that target is local/private. This supports SSH tunnels
// because the browser Host/Origin pair reflects the tunnel endpoint, while
// rejecting unrelated private-network origins.
func Validate(requestHost, origin string) bool {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return false
	}
	reqHost, reqPort := splitHostPortLoose(requestHost)
	originHost := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	reqHost = strings.ToLower(strings.TrimSuffix(reqHost, "."))
	if originHost != reqHost {
		return false
	}
	originPort := u.Port()
	if originPort == "" {
		if u.Scheme == "https" {
			originPort = "443"
		} else {
			originPort = "80"
		}
	}
	if reqPort != "" && reqPort != originPort {
		return false
	}
	if reqHost == "localhost" {
		return true
	}
	ip := net.ParseIP(reqHost)
	return ip != nil && (ip.IsLoopback() || isPrivate(ip))
}

func splitHostPortLoose(value string) (string, string) {
	value = strings.TrimSpace(value)
	if host, port, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]"), port
	}
	return strings.Trim(value, "[]"), ""
}

func isPrivate(ip net.IP) bool {
	if ip.IsPrivate() {
		return true
	}
	// Unique-local IPv6 for older Go/runtime variants.
	return len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc
}
