package desktop

import (
	"path/filepath"
	"testing"
)

func TestDeepLinkParser(t *testing.T) {
	cases := []struct {
		url          string
		wantResource string
		wantID       string
		wantErr      bool
	}{
		{"nexus://project/proj-123", "project", "proj-123", false},
		{"nexus://mission/msn-456", "mission", "msn-456", false},
		{"nexus://agent/agt-789?tab=terminal", "agent", "agt-789", false},
		{"http://project/123", "", "", true},
		{"nexus://unknown/123", "", "", true},
		{"nexus://project/", "", "", true},
	}

	for _, c := range cases {
		action, err := ParseDeepLink(c.url)
		if c.wantErr {
			if err == nil {
				t.Fatalf("url %s: expected error, got nil", c.url)
			}
			continue
		}
		if err != nil {
			t.Fatalf("url %s unexpected error: %v", c.url, err)
		}
		if action.Resource != c.wantResource || action.ID != c.wantID {
			t.Fatalf("url %s: got %+v, want resource=%s id=%s", c.url, action, c.wantResource, c.wantID)
		}
	}
}

func TestWindowStateManager(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "window-state.json")
	mgr := NewWindowStateManager(stateFile)

	initial := mgr.Load()
	if initial.Width < 640 || initial.Height < 480 {
		t.Fatalf("expected valid default window dimensions, got %+v", initial)
	}

	custom := WindowState{
		Width:     1440,
		Height:    900,
		X:         100,
		Y:         100,
		Maximized: true,
	}
	if err := mgr.Save(custom); err != nil {
		t.Fatalf("save window state failed: %v", err)
	}

	loaded := mgr.Load()
	if loaded.Width != 1440 || loaded.Height != 900 || !loaded.Maximized {
		t.Fatalf("loaded state does not match saved state: %+v", loaded)
	}
}

func TestCapabilities(t *testing.T) {
	app := NewApp(nil, nil)
	caps := app.GetCapabilities()
	if !caps.Native || !caps.FilePicker || !caps.FolderPicker || !caps.Notifications || !caps.Tray {
		t.Fatalf("desktop capabilities must declare native capabilities, got %+v", caps)
	}
}
