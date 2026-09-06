package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
)

// DefaultUpdateRegistryURL is the standard endpoint for update checks.
const DefaultUpdateRegistryURL = "https://updates.iapro.dev/v1/nexus/manifest.json"

// Service provides a unified update engine across CLI, Web, and Desktop.
type Service struct {
	mu          sync.RWMutex
	keyRing     *KeyRing
	registryURL string
	httpClient  *http.Client
	currentVer  string
	channel     string
	execPath    string
	method      InstallationMethod
	workTracker WorkTracker
}

// WorkTracker reports whether there is active execution that shouldn't be abruptly killed.
type WorkTracker interface {
	HasActiveWork() (bool, string)
}

// ServiceConfig configures the unified Update Service.
type ServiceConfig struct {
	RegistryURL string
	KeyRing     *KeyRing
	CurrentVer  string
	Channel     string
	ExecPath    string
	Method      InstallationMethod
	HTTPClient  *http.Client
	WorkTracker WorkTracker
}

// CheckResult details the availability and metadata of an update.
type CheckResult struct {
	CurrentVersion     string             `json:"current_version"`
	LatestVersion      string             `json:"latest_version"`
	Channel            string             `json:"channel"`
	InstallationMethod InstallationMethod `json:"installation_method"`
	UpdateAvailable    bool               `json:"update_available"`
	AllowsSelfUpdate   bool               `json:"allows_self_update"`
	Instruction        string             `json:"instruction,omitempty"`
	Changelog          string             `json:"changelog,omitempty"`
	Target             string             `json:"target,omitempty"`
	TargetArtifact     *Artifact          `json:"target_artifact,omitempty"`
	ActiveWork         bool               `json:"active_work"`
	ActiveWorkReason   string             `json:"active_work_reason,omitempty"`
}

// NewService creates a new unified Update Service.
func NewService(cfg ServiceConfig) *Service {
	ver := cfg.CurrentVer
	if ver == "" {
		ver = buildinfo.Version
	}
	ch := cfg.Channel
	if ch == "" {
		ch = "stable"
	}
	execP := cfg.ExecPath
	if execP == "" {
		if p, err := os.Executable(); err == nil {
			execP = p
		}
	}
	meth := cfg.Method
	if meth == "" {
		meth = DetectInstallationMethod(execP)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	regURL := cfg.RegistryURL
	if regURL == "" {
		if env := os.Getenv("NEXUS_UPDATE_REGISTRY_URL"); env != "" {
			regURL = env
		} else {
			regURL = DefaultUpdateRegistryURL
		}
	}
	kr := cfg.KeyRing
	if kr == nil {
		kr = NewKeyRing()
	}

	return &Service{
		keyRing:     kr,
		registryURL: regURL,
		httpClient:  client,
		currentVer:  ver,
		channel:     ch,
		execPath:    execP,
		method:      meth,
		workTracker: cfg.WorkTracker,
	}
}

// Check inspects the registry and evaluates update eligibility.
func (s *Service) Check(ctx context.Context) (*CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := &CheckResult{
		CurrentVersion:     s.currentVer,
		Channel:            s.channel,
		InstallationMethod: s.method,
		AllowsSelfUpdate:   s.method.AllowsSelfUpdate(),
		Instruction:        s.method.UpgradeInstruction(),
	}

	if s.workTracker != nil {
		hasWork, reason := s.workTracker.HasActiveWork()
		res.ActiveWork = hasWork
		res.ActiveWorkReason = reason
	}

	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	res.Target = target

	manifest, err := s.fetchManifest(ctx)
	if err != nil {
		res.LatestVersion = s.currentVer
		return res, err
	}

	res.LatestVersion = manifest.Version
	res.Changelog = manifest.Changelog

	policy := ManifestPolicy{
		Channel:        s.channel,
		CurrentVersion: s.currentVer,
		Target:         target,
		Now:            time.Now().UTC(),
	}

	if err := manifest.Validate(policy); err != nil {
		if errors.Is(err, ErrManifestDowngrade) {
			res.UpdateAvailable = false
			return res, nil
		}
		return res, fmt.Errorf("manifest validation failed: %w", err)
	}

	if compareVersions(manifest.Version, s.currentVer) > 0 {
		res.UpdateAvailable = true
		art := manifest.Artifacts[target]
		res.TargetArtifact = &art
	}

	return res, nil
}

// Apply downloads, verifies, and installs the update when permitted by InstallationMethod.
func (s *Service) Apply(ctx context.Context) (*Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.method.AllowsSelfUpdate() {
		return nil, fmt.Errorf("installation method %s does not allow self-update; use %s", s.method, s.method.UpgradeInstruction())
	}

	if s.workTracker != nil {
		hasWork, reason := s.workTracker.HasActiveWork()
		if hasWork {
			return nil, fmt.Errorf("cannot apply update: active work in progress (%s)", reason)
		}
	}

	manifest, err := s.fetchManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	policy := ManifestPolicy{
		Channel:        s.channel,
		CurrentVersion: s.currentVer,
		Target:         target,
		Now:            time.Now().UTC(),
	}

	if err := manifest.Validate(policy); err != nil {
		return nil, fmt.Errorf("manifest policy rejected: %w", err)
	}

	art, ok := manifest.Artifacts[target]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrManifestTarget, target)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, art.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid artifact URL: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading artifact failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifact download returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading artifact body failed: %w", err)
	}

	updater := NewUpdater(s.execPath, os.Getenv("HOME"))
	return updater.ApplyManifest(*manifest, policy, data)
}

func (s *Service) fetchManifest(ctx context.Context) (*Manifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.registryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching update manifest failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sig := resp.Header.Get("X-Signature-Ed25519")
	if sig == "" {
		sig = resp.Header.Get("X-Nexus-Signature")
	}

	if sig != "" {
		return s.keyRing.VerifyManifest(body, sig)
	}

	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
