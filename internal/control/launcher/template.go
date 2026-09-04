package launcher

import (
	"fmt"
	"strings"
)

// ResolveCommandTemplate expands a safe command template. Templates are tokenized
// directly (never through a shell), so a configured Agent cannot accidentally turn
// its workspace or prompt arguments into shell code. Supported placeholders are
// {cwd} and {args}; {args} must occupy a whole token and expands to all runtime args.
func ResolveCommandTemplate(template, cwd string, runtimeArgs []string) (string, []string, error) {
	tokens, err := splitCommandTemplate(template)
	if err != nil {
		return "", nil, err
	}
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("command template is empty")
	}
	var out []string
	seenArgs := false
	for _, token := range tokens {
		switch token {
		case "{args}":
			out = append(out, runtimeArgs...)
			seenArgs = true
		default:
			out = append(out, strings.ReplaceAll(token, "{cwd}", cwd))
		}
	}
	if !seenArgs {
		return "", nil, fmt.Errorf("command template must include {args} as its own argument")
	}
	if strings.TrimSpace(out[0]) == "" {
		return "", nil, fmt.Errorf("command template has an empty executable")
	}
	return out[0], out[1:], nil
}

// splitCommandTemplate is intentionally a small, non-shell grammar: spaces,
// quotes and backslash escapes are supported; shell operators are rejected.
func splitCommandTemplate(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n':
			flush()
		case '|', ';', '&', '<', '>', '`', '$':
			return nil, fmt.Errorf("command template cannot contain shell operators")
		default:
			current.WriteRune(r)
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("command template has an unfinished escape or quote")
	}
	flush()
	return tokens, nil
}
