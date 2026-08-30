package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFSBrowse(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "my-repo")
	_ = os.MkdirAll(filepath.Join(subDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module my-repo\n"), 0644)

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp FSBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CurrentPath != tempDir {
		t.Errorf("expected current path %s, got %s", tempDir, resp.CurrentPath)
	}

	if len(resp.Entries) == 0 {
		t.Fatalf("expected entries in %s, got 0", tempDir)
	}

	found := false
	for _, entry := range resp.Entries {
		if entry.Name == "my-repo" {
			found = true
			if !entry.IsDir {
				t.Errorf("expected my-repo to be dir")
			}
			if !entry.IsGit {
				t.Errorf("expected my-repo to be git repo")
			}
		}
	}
	if !found {
		t.Errorf("my-repo not found in browse results")
	}
}

func TestDetectTechReturnsEmptyArrayWhenNothingIsDetected(t *testing.T) {
	tech := detectTech(t.TempDir())

	if tech == nil {
		t.Fatal("expected an empty JSON-safe array, got nil")
	}
	if len(tech) != 0 {
		t.Fatalf("expected no detected technologies, got %v", tech)
	}
}

func TestFSInspect(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, ".git", "HEAD"), []byte("ref: refs/heads/feature-xyz\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name":"test-app"}`), 0644)

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/inspect?path="+tempDir, nil)
	w := httptest.NewRecorder()

	handler.handleFSInspect(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp FSInspectResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Exists || !resp.IsDir || !resp.IsGit {
		t.Errorf("expected exists=true, isDir=true, isGit=true; got %+v", resp)
	}

	if resp.GitBranch != "feature-xyz" {
		t.Errorf("expected branch feature-xyz, got %s", resp.GitBranch)
	}
}

func TestFSMkdir(t *testing.T) {
	tempDir := t.TempDir()
	newFolder := filepath.Join(tempDir, "created-folder")

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	body := `{"path":"` + newFolder + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/mkdir", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.handleFSMkdir(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(newFolder); err != nil {
		t.Fatalf("expected folder to exist on disk: %v", err)
	}
}

func TestGetGitRemoteRedactsCredentials(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("[remote \"origin\"]\n\turl = https://secret-token@github.com/acme/private.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := getGitRemote(dir); got != "https://github.com/acme/private.git" {
		t.Fatalf("remote = %q, credentials must be redacted", got)
	}
}

func TestFilesystemRootUsesCurrentVolumeInsteadOfUnixLiteral(t *testing.T) {
	root := filesystemRoot(t.TempDir())
	if root == "" {
		t.Fatal("filesystem root must not be empty")
	}
	if filepath.Dir(root) != root {
		t.Fatalf("filesystem root %q must be a root directory", root)
	}
}

func TestDefaultScanRootsAreDerivedFromUserPaths(t *testing.T) {
	home := t.TempDir()
	cwd := filepath.Join(home, "work")
	roots := defaultScanRoots(home, cwd)
	for _, root := range roots {
		if root == "/projetos" {
			t.Fatal("default scan roots must not contain a Unix-only /projetos literal")
		}
	}
	if len(roots) == 0 {
		t.Fatal("expected at least one derived scan root")
	}
}
