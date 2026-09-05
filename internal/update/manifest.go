package update

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Artifact struct {
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature,omitempty"`
}

type Manifest struct {
	SchemaVersion     int                 `json:"schema_version"`
	Channel           string              `json:"channel"`
	Version           string              `json:"version"`
	ReleaseDate       string              `json:"release_date"`
	ExpiresAt         string              `json:"expires_at,omitempty"`
	MinNexusVersion   string              `json:"min_nexus_version,omitempty"`
	MinMaestroVersion string              `json:"min_maestro_version,omitempty"`
	KeyID             string              `json:"key_id"`
	Changelog         string              `json:"changelog,omitempty"`
	Artifacts         map[string]Artifact `json:"artifacts,omitempty"`
}

type ManifestPolicy struct {
	Channel        string
	CurrentVersion string
	NexusVersion   string
	MaestroVersion string
	Target         string
	Now            time.Time
}

var (
	ErrManifestExpired       = errors.New("update manifest has expired")
	ErrManifestChannel       = errors.New("update manifest channel is not allowed")
	ErrManifestDowngrade     = errors.New("update manifest would downgrade the installed version")
	ErrManifestCompatibility = errors.New("installed version is incompatible with update manifest")
	ErrManifestTarget        = errors.New("update manifest has no artifact for target")
	ErrManifestArtifact      = errors.New("update manifest artifact is invalid")
)

// Validate applies local policy before an artifact can be downloaded or installed.
// Empty policy fields intentionally skip that dimension for backwards-compatible
// manifest inspection; callers that install updates should always provide them.
func (m Manifest) Validate(policy ManifestPolicy) error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema version: %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Channel) == "" || strings.TrimSpace(m.Version) == "" {
		return errors.New("manifest channel and version are required")
	}
	if policy.Channel != "" && !strings.EqualFold(policy.Channel, m.Channel) {
		return fmt.Errorf("%w: expected %s, got %s", ErrManifestChannel, policy.Channel, m.Channel)
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if m.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, m.ExpiresAt)
		if err != nil {
			return fmt.Errorf("invalid expires_at: %w", err)
		}
		if !expires.After(now) {
			return ErrManifestExpired
		}
	}
	if policy.CurrentVersion != "" && compareVersions(m.Version, policy.CurrentVersion) < 0 {
		return fmt.Errorf("%w: installed=%s manifest=%s", ErrManifestDowngrade, policy.CurrentVersion, m.Version)
	}
	if policy.NexusVersion != "" && m.MinNexusVersion != "" && compareVersions(policy.NexusVersion, m.MinNexusVersion) < 0 {
		return fmt.Errorf("%w: Nexus %s requires %s", ErrManifestCompatibility, m.MinNexusVersion, policy.NexusVersion)
	}
	if policy.MaestroVersion != "" && m.MinMaestroVersion != "" && compareVersions(policy.MaestroVersion, m.MinMaestroVersion) < 0 {
		return fmt.Errorf("%w: Maestro %s requires %s", ErrManifestCompatibility, m.MinMaestroVersion, policy.MaestroVersion)
	}
	if policy.Target != "" {
		artifact, ok := m.Artifacts[policy.Target]
		if !ok {
			return fmt.Errorf("%w: %s", ErrManifestTarget, policy.Target)
		}
		if artifact.Size < 0 || len(artifact.SHA256) != 64 {
			return fmt.Errorf("%w: %s", ErrManifestArtifact, policy.Target)
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return fmt.Errorf("%w: %s checksum: %v", ErrManifestArtifact, policy.Target, err)
		}
	}
	return nil
}

func compareVersions(a, b string) int {
	parse := func(v string) []int {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		parts := strings.SplitN(v, "-", 2)[0]
		segments := strings.Split(parts, ".")
		values := make([]int, 3)
		for i := 0; i < len(values) && i < len(segments); i++ {
			values[i], _ = strconv.Atoi(segments[i])
		}
		return values
	}
	left, right := parse(a), parse(b)
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}
