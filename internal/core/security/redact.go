package security

import (
	"regexp"
	"strings"
)

var (
	bearerRegex    = regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9_\-\.]{10,}`)
	authHeaderRegex= regexp.MustCompile(`(?i)(authorization:\s*)([^\r\n]+)`)
	jwtRegex       = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]+`)
	apiKeyRegex    = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|auth_token)\s*[:=]\s*["']?([a-zA-Z0-9_\-\.]{8,})["']?`)
	privateKeyRegex= regexp.MustCompile(`(?s)-----BEGIN[ A-Z0-9_-]*PRIVATE KEY-----.*?-----END[ A-Z0-9_-]*PRIVATE KEY-----`)
	cookieRegex    = regexp.MustCompile(`(?i)(cookie:\s*)([^\r\n]+)`)
	openaiKeyRegex = regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`)
	anthropicRegex = regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-_]{20,}`)
	googleOAuthReg = regexp.MustCompile(`ya29\.[a-zA-Z0-9_\-]{20,}`)
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
	res = bearerRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = authHeaderRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = cookieRegex.ReplaceAllString(res, "${1}[REDACTED]")
	res = apiKeyRegex.ReplaceAllStringFunc(res, func(match string) string {
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
