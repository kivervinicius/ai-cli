package desktop

import (
	"errors"
	"net/url"
	"strings"
)

// DeepLinkAction identifies the destination and resource targeted by a nexus:// URL.
type DeepLinkAction struct {
	Resource string            `json:"resource"` // "project", "mission", "agent"
	ID       string            `json:"id"`
	Params   map[string]string `json:"params,omitempty"`
}

// ParseDeepLink parses and validates a nexus:// URI.
// Supported schemes:
//   nexus://project/<id>
//   nexus://mission/<id>
//   nexus://agent/<id>
func ParseDeepLink(rawURL string) (*DeepLinkAction, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "nexus" {
		return nil, errors.New("invalid scheme: expected nexus://")
	}

	resource := u.Host
	path := strings.Trim(u.Path, "/")

	if resource == "" && path != "" {
		parts := strings.Split(path, "/")
		resource = parts[0]
		if len(parts) > 1 {
			path = parts[1]
		}
	}

	switch resource {
	case "project", "mission", "agent":
		if path == "" {
			return nil, errors.New("missing resource id in deep link")
		}
	default:
		return nil, errors.New("unsupported deep link resource: " + resource)
	}

	params := make(map[string]string)
	for k, v := range u.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	return &DeepLinkAction{
		Resource: resource,
		ID:       path,
		Params:   params,
	}, nil
}
