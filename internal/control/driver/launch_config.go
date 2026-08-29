package driver

import (
	"fmt"
	"sort"
	"strings"
)

// ApplyLaunchConfiguration converts the subset of Agent configuration that is
// truthfully supported by provider CLIs into command arguments. Unsupported
// values return an error instead of being silently ignored.
func ApplyLaunchConfiguration(provider, model string, options map[string]any, baseArgs []string) ([]string, error) {
	args := append([]string(nil), baseArgs...)
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.TrimSpace(model)
	if model != "" {
		switch provider {
		case "codex", "claude", "gemini", "opencode":
			args = append(args, "--model", model)
		case "fake":
			args = append(args, "--model", model)
		default:
			return nil, fmt.Errorf("provider %q does not declare model override support", provider)
		}
	}
	if len(options) == 0 {
		return args, nil
	}

	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := options[key]
		switch key {
		case "extra_args":
			extra, err := stringSliceOption(value)
			if err != nil {
				return nil, fmt.Errorf("provider option extra_args: %w", err)
			}
			args = append(args, extra...)
		default:
			return nil, fmt.Errorf("provider option %q is not supported by the supervised launcher", key)
		}
	}
	return args, nil
}

func stringSliceOption(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("must contain non-empty strings")
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("must be an array of strings")
	}
}
