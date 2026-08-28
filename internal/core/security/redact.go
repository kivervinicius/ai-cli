package security

import (
	"regexp"
	"strings"
)

var (
	bearerRegex     = regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9_\-\.]{10,}`)
	authHeaderRegex = regexp.MustCompile(`(?i)(authorization:\s*)([^\r\n]+)`)
	jwtRegex        = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]+`)
	apiKeyRegex     = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|auth_token|passwd|credentials?)\s*[:=]\s*["']?([^\s"';,\r\n]{6,})["']?`)
	privateKeyRegex = regexp.MustCompile(`(?s)-----BEGIN[ A-Z0-9_-]*PRIVATE KEY-----.*?-----END[ A-Z0-9_-]*PRIVATE KEY-----`)
	cookieRegex     = regexp.MustCompile(`(?i)(cookie:\s*)([^\r\n]+)`)
	openaiKeyRegex  = regexp.MustCompile(`sk-(proj-|admin-)?[a-zA-Z0-9_\-]{20,}`)
	anthropicRegex  = regexp.MustCompile(`sk-ant-[a-zA-Z0-9_\-]{20,}`)
	googleOAuthReg   = regexp.MustCompile(`ya29\.[a-zA-Z0-9_\-]{20,}`)
	githubTokenReg   = regexp.MustCompile(`(gh[pousr]_[a-zA-Z0-9]{20,}|github_pat_[a-zA-Z0-9_]{20,})`)
	awsKeyRegex      = regexp.MustCompile(`(AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}`)
	uriPasswordRegex = regexp.MustCompile(`(?i)([a-z0-9+.-]+://[^:\s]+:)(.*)(@[^/@\s]+(?:/[^\s]*)?)`)
)

// Redact removes secrets, OAuth tokens, API keys, and sensitive headers from strings.
func Redact(input string) string {
	if input == "" {
		return ""
	}
	res := privateKeyRegex.ReplaceAllString(input, "[REDACTED_PRIVATE_KEY]")
	res = jwtRegex.ReplaceAllString(res, "[REDACTED_JWT_TOKEN]")
	res = openaiKeyRegex.ReplaceAllString(res, "[REDACTED_OPENAI_KEY]")
	res = anthropicRegex.ReplaceAllString(res, "[REDACTED_ANTHROPIC_KEY]")
	res = googleOAuthReg.ReplaceAllString(res, "[REDACTED_GOOGLE_TOKEN]")
	res = githubTokenReg.ReplaceAllString(res, "[REDACTED_GITHUB_TOKEN]")
	res = awsKeyRegex.ReplaceAllString(res, "[REDACTED_AWS_KEY]")
	res = uriPasswordRegex.ReplaceAllString(res, "${1}[REDACTED]${3}")
	res = bearerRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = authHeaderRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = cookieRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = apiKeyRegex.ReplaceAllStringFunc(res, func(match string) string {
		if strings.Contains(match, "[REDACTED") {
			return match
		}
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 {
			return parts[0] + ": [REDACTED]"
		}
		parts = strings.SplitN(match, "=", 2)
		if len(parts) == 2 {
			return parts[0] + "=[REDACTED]"
		}
		return "[REDACTED_SECRET]"
	})
	return res
}

// RedactSlice returns a copy of the slice with all string elements redacted.
func RedactSlice(items []string) []string {
	if items == nil {
		return nil
	}
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = Redact(item)
	}
	return out
}
