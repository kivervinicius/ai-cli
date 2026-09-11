package contextsnapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/security"
)

const (
	ScannerVersion         = "project-intelligence-v1"
	MaxDiscoveryDepth      = 6
	MaxDiscoveryCandidates = 512
	MaxDiscoveryFileBytes  = 256 * 1024
	MaxDiscoveryTotalBytes = 4 * 1024 * 1024
	MaxProjectFacts        = 500
)

type FactValueType string

const (
	FactString FactValueType = "STRING"
	FactList   FactValueType = "LIST"
	FactMap    FactValueType = "MAP"
	FactBool   FactValueType = "BOOL"
)

type FactBasis string

const (
	FactObserved FactBasis = "OBSERVED"
	FactDerived  FactBasis = "DERIVED"
)

type FactConfidence string

const (
	ConfidenceHigh   FactConfidence = "HIGH"
	ConfidenceMedium FactConfidence = "MEDIUM"
	ConfidenceLow    FactConfidence = "LOW"
)

type FactProvenance struct {
	SourcePath string    `json:"source"`
	Locator    string    `json:"locator,omitempty"`
	Extractor  string    `json:"extractor"`
	Digest     string    `json:"digest,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type ProjectFact struct {
	Category   string           `json:"category"`
	Key        string           `json:"key"`
	ValueType  FactValueType    `json:"value_type"`
	Value      any              `json:"value"`
	Basis      FactBasis        `json:"basis"`
	Confidence FactConfidence   `json:"confidence"`
	ObservedAt time.Time        `json:"observed_at"`
	Provenance []FactProvenance `json:"provenance"`
}

type SnapshotCompleteness string

const (
	CompletenessComplete SnapshotCompleteness = "COMPLETE"
	CompletenessPartial  SnapshotCompleteness = "PARTIAL"
)

// ProjectContextSnapshot is an immutable, structured view of static project
// evidence. Values are intentionally JSON-compatible so the same contract can
// be persisted and consumed by Planning, Composer, and future routers.
type ProjectContextSnapshot struct {
	ID             string               `json:"id,omitempty"`
	ProjectID      string               `json:"project_id,omitempty"`
	CanonicalPath  string               `json:"canonical_path,omitempty"`
	ScannerVersion string               `json:"scanner_version"`
	Identity       CodeIdentity         `json:"identity"`
	Completeness   SnapshotCompleteness `json:"completeness"`
	Facts          []ProjectFact        `json:"facts"`
	Warnings       []string             `json:"warnings,omitempty"`
	ObservedAt     time.Time            `json:"observed_at"`
}

type discoveryState struct {
	root       string
	observedAt time.Time
	candidates int
	totalBytes int64
	facts      []ProjectFact
	warnings   []string
	seen       map[string]bool
}

func recognizedDocument(path string) bool {
	base := filepath.Base(path)
	if base == "go.mod" || base == "go.work" || base == "package.json" || base == "pnpm-workspace.yaml" || base == "yarn.lock" || base == "package-lock.json" || base == "Cargo.toml" || base == "pyproject.toml" || base == "requirements.txt" || base == "Makefile" || base == "Dockerfile" || base == "README.md" || base == "AGENTS.md" {
		return true
	}
	slash := filepath.ToSlash(path)
	return strings.HasPrefix(slash, ".github/workflows/") || strings.HasPrefix(slash, ".gitlab/") || strings.HasPrefix(slash, "DEV/") && strings.HasSuffix(base, ".md")
}

func addFact(state *discoveryState, category, key string, valueType FactValueType, value any, confidence FactConfidence, source, locator, extractor string, digest string) {
	if len(state.facts) >= MaxProjectFacts {
		state.warnings = append(state.warnings, "project fact budget exceeded")
		return
	}
	identity := category + "\x00" + key
	if state.seen[identity] {
		return
	}
	state.seen[identity] = true
	state.facts = append(state.facts, ProjectFact{Category: category, Key: key, ValueType: valueType, Value: value, Basis: FactObserved, Confidence: confidence, ObservedAt: state.observedAt, Provenance: []FactProvenance{{SourcePath: filepath.ToSlash(source), Locator: locator, Extractor: extractor, Digest: digest, ObservedAt: state.observedAt}}})
}

func readDiscoveryFile(state *discoveryState, path string) ([]byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return nil, "", fmt.Errorf("unsafe source")
	}
	if info.Size() > MaxDiscoveryFileBytes {
		state.warnings = append(state.warnings, fmt.Sprintf("skipped oversized file %s", filepath.ToSlash(path)))
		return nil, "", nil
	}
	if state.totalBytes+info.Size() > MaxDiscoveryTotalBytes {
		state.warnings = append(state.warnings, "project discovery byte budget exceeded")
		return nil, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	state.totalBytes += int64(len(data))
	return data, digestBytes(data), nil
}

func parseManifest(state *discoveryState, relative string, data []byte, digest string) {
	base := filepath.Base(relative)
	source := filepath.ToSlash(relative)
	switch base {
	case "go.mod":
		module := ""
		goVersion := ""
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[0] == "module" {
				module = fields[1]
			}
			if len(fields) >= 2 && fields[0] == "go" {
				goVersion = fields[1]
			}
		}
		if module != "" {
			addFact(state, "stack", "go.module", FactString, module, ConfidenceHigh, source, "module", "go.mod", digest)
		}
		if goVersion != "" {
			addFact(state, "stack", "go.version", FactString, goVersion, ConfidenceHigh, source, "go", "go.mod", digest)
		}
		addFact(state, "stack", "go", FactBool, true, ConfidenceHigh, source, "", "go.mod", digest)
	case "package.json":
		var manifest struct {
			Name            string                     `json:"name"`
			PackageManager  string                     `json:"packageManager"`
			Scripts         map[string]json.RawMessage `json:"scripts"`
			Dependencies    map[string]json.RawMessage `json:"dependencies"`
			DevDependencies map[string]json.RawMessage `json:"devDependencies"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			state.warnings = append(state.warnings, "invalid package.json")
			return
		}
		if manifest.Name != "" {
			addFact(state, "stack", "node.package", FactString, manifest.Name, ConfidenceHigh, source, "name", "package.json", digest)
		}
		if manifest.PackageManager != "" {
			addFact(state, "dependencies", "package_manager", FactString, manifest.PackageManager, ConfidenceHigh, source, "packageManager", "package.json", digest)
		}
		scripts := make([]string, 0, len(manifest.Scripts))
		for name := range manifest.Scripts {
			scripts = append(scripts, name)
		}
		sort.Strings(scripts)
		if len(scripts) > 0 {
			addFact(state, "scripts", "npm", FactList, scripts, ConfidenceHigh, source, "scripts", "package.json", digest)
		}
		deps := make([]string, 0, len(manifest.Dependencies)+len(manifest.DevDependencies))
		for name := range manifest.Dependencies {
			deps = append(deps, name)
		}
		for name := range manifest.DevDependencies {
			deps = append(deps, name)
		}
		sort.Strings(deps)
		if len(deps) > 0 {
			addFact(state, "dependencies", "node", FactList, deps, ConfidenceMedium, source, "dependencies", "package.json", digest)
		}
	case "Cargo.toml":
		addFact(state, "stack", "rust", FactBool, true, ConfidenceHigh, source, "", "Cargo.toml", digest)
	case "pyproject.toml", "requirements.txt":
		addFact(state, "stack", "python", FactBool, true, ConfidenceHigh, source, "", base, digest)
	case "Makefile":
		addFact(state, "scripts", "make", FactBool, true, ConfidenceHigh, source, "", "Makefile", digest)
	case "Dockerfile":
		addFact(state, "architecture", "containerized", FactBool, true, ConfidenceMedium, source, "", "Dockerfile", digest)
	}
}

func discoverPath(state *discoveryState, path string, entry os.DirEntry, root string) error {
	if state.candidates >= MaxDiscoveryCandidates {
		state.warnings = append(state.warnings, "project discovery candidate budget exceeded")
		return filepath.SkipAll
	}
	state.candidates++
	if entry.IsDir() {
		return nil
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	source := filepath.ToSlash(relative)
	base := filepath.Base(relative)
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") || strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.tsx") {
		addFact(state, "tests", "present", FactBool, true, ConfidenceHigh, source, "", "filename-pattern", "")
	}
	if base == "main.go" || base == "main.ts" || base == "main.tsx" || base == "index.html" {
		addFact(state, "entrypoints", source, FactString, source, ConfidenceMedium, source, "filename", "filename-pattern", "")
	}
	if strings.HasPrefix(source, ".github/workflows/") || strings.HasPrefix(source, ".gitlab/") {
		addFact(state, "ci", source, FactBool, true, ConfidenceHigh, source, "workflow", "path-pattern", "")
	}
	if base == "AGENTS.md" || strings.HasPrefix(source, "DEV/") {
		addFact(state, "conventions", source, FactBool, true, ConfidenceMedium, source, "", "durable-doc", "")
	}
	if !recognizedDocument(relative) {
		return nil
	}
	data, digest, err := readDiscoveryFile(state, path)
	if err != nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	parseManifest(state, relative, []byte(security.Redact(string(data))), digest)
	return nil
}

func classifyDirectory(state *discoveryState, path, root string) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return
	}
	source := filepath.ToSlash(relative)
	base := filepath.Base(relative)
	if base == "cmd" || base == "internal" || base == "src" || base == "web" || base == "packages" || base == "modules" {
		addFact(state, "architecture", "module."+source, FactBool, true, ConfidenceMedium, source, "directory", "directory-pattern", "")
	}
}

// Discover performs bounded, static project discovery. It never executes a
// project command and returns PARTIAL with warnings when budgets are reached.
func Discover(root string, metadata Metadata) (ProjectContextSnapshot, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return ProjectContextSnapshot{}, fmt.Errorf("project root is required")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return ProjectContextSnapshot{}, fmt.Errorf("project root unavailable")
	}
	identity, err := InspectCodeIdentity(root)
	if err != nil {
		return ProjectContextSnapshot{}, err
	}
	now := time.Now().UTC()
	state := &discoveryState{root: root, observedAt: now, seen: map[string]bool{}}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			state.warnings = append(state.warnings, walkErr.Error())
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == ".cache") {
				return filepath.SkipDir
			}
			depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
			if depth > MaxDiscoveryDepth {
				return filepath.SkipDir
			}
			classifyDirectory(state, path, root)
			return nil
		}
		return discoverPath(state, path, entry, root)
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return ProjectContextSnapshot{}, walkErr
	}
	sort.Slice(state.facts, func(i, j int) bool {
		if state.facts[i].Category == state.facts[j].Category {
			return state.facts[i].Key < state.facts[j].Key
		}
		return state.facts[i].Category < state.facts[j].Category
	})
	completeness := CompletenessComplete
	if len(state.warnings) > 0 || state.candidates >= MaxDiscoveryCandidates || state.totalBytes >= MaxDiscoveryTotalBytes {
		completeness = CompletenessPartial
	}
	return ProjectContextSnapshot{ProjectID: metadata.ProjectID, CanonicalPath: root, ScannerVersion: ScannerVersion, Identity: identity, Completeness: completeness, Facts: state.facts, Warnings: state.warnings, ObservedAt: now}, nil
}
