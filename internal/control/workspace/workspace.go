package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
)

// Project represents a managed developer workspace/repository.
type Project struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// Store manages persistent projects across restarts.
type Store struct {
	mu       sync.RWMutex
	filePath string
	projects map[string]Project
}

var (
	defaultStore *Store
	storeOnce    sync.Once
)

// DefaultStore returns the global workspace/project store.
func DefaultStore() *Store {
	storeOnce.Do(func() {
		dataDir, err := config.DataDir()
		if err != nil {
			dataDir = filepath.Join(os.TempDir(), "ai-control")
		}
		_ = os.MkdirAll(dataDir, 0700)
		defaultStore = NewStore(filepath.Join(dataDir, "projects.json"))
	})
	return defaultStore
}

// NewStore initializes a Project store with persistence.
func NewStore(filePath string) *Store {
	s := &Store{
		filePath: filePath,
		projects: make(map[string]Project),
	}
	_ = s.load()

	// Ensure current working directory is always present
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		_ = s.ensureCwd(cwd)
	}

	return s
}

func (s *Store) ensureCwd(cwd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := filepath.Clean(cwd)
	for _, p := range s.projects {
		if filepath.Clean(p.Path) == clean {
			return nil
		}
	}

	name := filepath.Base(clean)
	id := makeID(name)
	s.projects[clean] = Project{
		ID:         id,
		Name:       name,
		Path:       clean,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
	return s.saveLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var list []Project
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, p := range list {
		if p.Path != "" {
			s.projects[filepath.Clean(p.Path)] = p
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	var list []Project
	for _, p := range s.projects {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.filePath)
}

// List returns all registered projects sorted with most recently used first.
func (s *Store) List() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Project
	for _, p := range s.projects {
		result = append(result, p)
	}
	return result
}

// Add registers a new project path.
func (s *Store) Add(path, name string) (Project, error) {
	if strings.TrimSpace(path) == "" {
		return Project{}, fmt.Errorf("project path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = filepath.Clean(path)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return Project{}, fmt.Errorf("path does not exist: %w", err)
	}
	if !fi.IsDir() {
		return Project{}, fmt.Errorf("path is not a directory: %s", absPath)
	}

	if name == "" {
		name = filepath.Base(absPath)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clean := filepath.Clean(absPath)
	p := Project{
		ID:         makeID(name),
		Name:       name,
		Path:       clean,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	s.projects[clean] = p
	if err := s.saveLocked(); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Remove deletes a project from management.
func (s *Store) Remove(idOrPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetKey := ""
	for key, p := range s.projects {
		if p.ID == idOrPath || p.Path == idOrPath || filepath.Clean(p.Path) == filepath.Clean(idOrPath) {
			targetKey = key
			break
		}
	}

	if targetKey == "" {
		return fmt.Errorf("project %q not found", idOrPath)
	}

	delete(s.projects, targetKey)
	return s.saveLocked()
}

// Touch updates the last used timestamp of a project.
func (s *Store) Touch(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := filepath.Clean(path)
	if p, ok := s.projects[clean]; ok {
		p.LastUsedAt = time.Now()
		s.projects[clean] = p
		_ = s.saveLocked()
	}
}

func makeID(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	var sb strings.Builder
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else if r == ' ' || r == '_' {
			sb.WriteRune('-')
		}
	}
	id := sb.String()
	if id == "" {
		id = fmt.Sprintf("proj-%d", time.Now().UnixNano()%100000)
	}
	return id
}
