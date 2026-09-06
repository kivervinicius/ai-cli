package originpolicy

import (
	"net"
	"net/url"
	"strings"
)

// IsDesktopOrigin returns true if the origin is generated strictly by the native desktop shell (Wails).
func IsDesktopOrigin(u *url.URL) bool {
	if u == nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "wails" && scheme != "http" && scheme != "https" {
		return false
	}
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	return hostname == "wails" || hostname == "wails.localhost"
}

func isDesktopOrigin(u *url.URL) bool {
	return IsDesktopOrigin(u)
}

// IsTrustedDesktopRequest verifies if the request comes strictly from the Wails native desktop shell
// and targets loopback or the internal wails host.
func IsTrustedDesktopRequest(requestHost, originHeader, refererHeader string) bool {
	reqHost, _ := splitHostPortLoose(requestHost)
	reqHost = strings.ToLower(strings.TrimSuffix(reqHost, "."))

	isLoopback := reqHost == "wails" || reqHost == "localhost"
	if !isLoopback {
		ip := net.ParseIP(reqHost)
		isLoopback = ip != nil && ip.IsLoopback()
	}
	if !isLoopback {
		return false
	}

	originHeader = strings.TrimSpace(originHeader)
	refererHeader = strings.TrimSpace(refererHeader)

	// If Origin is provided, it is authoritative and MUST be valid.
	if originHeader != "" {
		u, err := url.Parse(originHeader)
		return err == nil && IsDesktopOrigin(u)
	}

	// If Origin is absent, fallback to Referer which MUST be valid.
	if refererHeader != "" {
		u, err := url.Parse(refererHeader)
		return err == nil && IsDesktopOrigin(u)
	}

	return false
}

// Validate accepts browser origins only when they target the exact HTTP Host
// of the request and that target is local/private. This supports SSH tunnels
// because the browser Host/Origin pair reflects the tunnel endpoint, while
// rejecting unrelated private-network origins.
func Validate(requestHost, origin string) bool {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return false
	}
	reqHost, reqPort := splitHostPortLoose(requestHost)
	reqHost = strings.ToLower(strings.TrimSuffix(reqHost, "."))

	// Allow native desktop WebView origins (Wails) targeting internal loopback or wails host.
	if isDesktopOrigin(u) {
		if reqHost == "wails" || reqHost == "localhost" {
			return true
		}
		ip := net.ParseIP(reqHost)
		return ip != nil && ip.IsLoopback()
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return false
	}
	originHost := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
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
