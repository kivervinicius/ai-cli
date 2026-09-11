package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// scopeRecord is deliberately kept separate from profile.json so old profile
// metadata remains readable and legacy snapshots can be quarantined.
type scopeRecord struct {
	AccountID       string `json:"account_id"`
	IdentityVersion string `json:"identity_version"`
	Identity        string `json:"identity,omitempty"`
}

var scopeMu sync.Mutex

// AccountScope returns the stable scope for a registered profile. An empty
// identity never upgrades an unknown account into an attributed observation.
// When a verified identity changes, a new identity version is created.
func AccountScope(provider, profileName, authenticatedIdentity string) (model.AccountScope, error) {
	scopeMu.Lock()
	defer scopeMu.Unlock()
	root, err := config.ProfileRoot(provider, profileName)
	if err != nil {
		return model.AccountScope{}, err
	}
	path := filepath.Join(root, "account-scope.json")
	var record scopeRecord
	if data, readErr := os.ReadFile(path); readErr == nil {
		if err := json.Unmarshal(data, &record); err != nil {
			return model.AccountScope{}, fmt.Errorf("read account scope: %w", err)
		}
	}
	identity := strings.TrimSpace(authenticatedIdentity)
	if record.AccountID == "" {
		var idErr error
		record.AccountID, idErr = randomID()
		if idErr != nil {
			return model.AccountScope{}, idErr
		}
		record.IdentityVersion = "1"
	}
	if identity != "" && record.Identity != "" && !strings.EqualFold(identity, record.Identity) {
		record.IdentityVersion = incrementVersion(record.IdentityVersion)
	}
	if identity != "" {
		record.Identity = identity
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return model.AccountScope{}, err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return model.AccountScope{}, err
	}
	// Keep the durable local account id for future registration, but do not
	// expose an attributable scope until an authenticated identity is known.
	identityVersion := ""
	if record.Identity != "" {
		identityVersion = record.IdentityVersion
	}
	return model.AccountScope{ProviderID: provider, ProfileID: profileName, AccountID: record.AccountID, IdentityVersion: identityVersion}, nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate account scope id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func incrementVersion(version string) string {
	var n int
	if _, err := fmt.Sscanf(version, "%d", &n); err != nil || n < 1 {
		n = 1
	}
	return fmt.Sprintf("%d", n+1)
}
