package profile

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AccountInfo struct {
	Email         string       `json:"email"`
	Plan          string       `json:"plan"`
	Status        string       `json:"status"`
	QuotaSummary  string       `json:"quota_summary"`
	Limits        []string     `json:"limits"`
	Quota         QuotaDetails `json:"quota"`
	ExpiresAt     time.Time    `json:"expires_at,omitempty"`
	Authenticated bool         `json:"authenticated"`
}

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+`)

// GetAccountInfo extracts non-secret account identity, email, and subscription plan.
func GetAccountInfo(provider, name string) AccountInfo {
	home, err := Home(provider, name)
	if err != nil {
		return AccountInfo{Status: "Error", Plan: "Desconhecido"}
	}

	var info AccountInfo
	switch provider {
	case "agy":
		info = getAgyAccountInfo(home, name)
	case "codex":
		info = getCodexAccountInfo(home, name)
	default:
		info = AccountInfo{Status: "Provider desconhecido"}
	}

	info.Quota = GetQuotaDetails(provider, name, info.Plan, info.Email)
	return info
}

func getAgyAccountInfo(home, profileName string) AccountInfo {
	info := AccountInfo{
		Plan:          "Google AI Pro",
		Status:        "Autenticado",
		Authenticated: true,
	}

	// 1. Check google_accounts.json in profile home
	accountsFile := filepath.Join(home, ".gemini", "google_accounts.json")
	if data, err := os.ReadFile(accountsFile); err == nil {
		var acc struct {
			Active string `json:"active"`
		}
		if json.Unmarshal(data, &acc) == nil && acc.Active != "" {
			info.Email = acc.Active
		}
	}

	// 2. Check jetski_state.pbtxt
	if info.Email == "" {
		jetskiFile := filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
		if data, err := os.ReadFile(jetskiFile); err == nil && len(data) > 0 {
			matches := emailRegex.FindAllString(string(data), -1)
			if len(matches) > 0 {
				info.Email = matches[0]
			}
		}
	}

	// 3. Fallback to clean name formatting
	if info.Email == "" {
		if strings.Contains(profileName, "@") {
			info.Email = profileName
		} else {
			info.Email = profileName + "@gmail.com"
		}
	}

	return info
}

func getCodexAccountInfo(home, profileName string) AccountInfo {
	info := AccountInfo{
		Plan:          "ChatGPT Plus",
		Status:        "Autenticado",
		Authenticated: true,
	}

	// 1. Read auth.json
	authFile := filepath.Join(home, "auth.json")
	if data, err := os.ReadFile(authFile); err == nil {
		var authData struct {
			Tokens struct {
				IDToken string `json:"id_token"`
			} `json:"tokens"`
		}
		if json.Unmarshal(data, &authData) == nil && authData.Tokens.IDToken != "" {
			parts := strings.Split(authData.Tokens.IDToken, ".")
			if len(parts) >= 2 {
				payloadSeg := parts[1]
				if pad := len(payloadSeg) % 4; pad != 0 {
					payloadSeg += strings.Repeat("=", 4-pad)
				}
				rawPayload, err := base64.URLEncoding.DecodeString(payloadSeg)
				if err == nil {
					var claims struct {
						Email      string `json:"email"`
						OpenAIAuth struct {
							PlanType    string `json:"chatgpt_plan_type"`
							ActiveUntil string `json:"chatgpt_subscription_active_until"`
						} `json:"https://api.openai.com/auth"`
					}
					if json.Unmarshal(rawPayload, &claims) == nil {
						if claims.Email != "" {
							info.Email = claims.Email
						}
						if claims.OpenAIAuth.PlanType != "" {
							info.Plan = "ChatGPT " + strings.Title(strings.ToLower(claims.OpenAIAuth.PlanType))
						}
						if claims.OpenAIAuth.ActiveUntil != "" {
							if t, err := time.Parse(time.RFC3339, claims.OpenAIAuth.ActiveUntil); err == nil {
								info.ExpiresAt = t
							}
						}
					}
				}
			}
		}
	}

	// 2. Fallback to clean name formatting
	if info.Email == "" {
		if strings.Contains(profileName, "@") {
			info.Email = profileName
		} else {
			info.Email = profileName + "@openai.com"
		}
	}

	return info
}
