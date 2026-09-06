package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// WindowState persists the window geometry and layout state across restarts.
type WindowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Maximized bool `json:"maximized"`
}

// WindowStateManager safely loads and saves window geometry.
type WindowStateManager struct {
	mu       sync.RWMutex
	filePath string
}

// NewWindowStateManager initializes a WindowStateManager with a given state file path.
func NewWindowStateManager(filePath string) *WindowStateManager {
	if filePath == "" {
		home, _ := os.UserHomeDir()
		filePath = filepath.Join(home, ".local", "share", "iapro-nexus", "window-state.json")
	}
	return &WindowStateManager{filePath: filePath}
}

// Load reads the window state, returning sensible defaults if missing or corrupted.
func (m *WindowStateManager) Load() WindowState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	defaultState := WindowState{
		Width:     1280,
		Height:    800,
		Maximized: false,
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return defaultState
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultState
	}

	if state.Width < 640 {
		state.Width = 1280
	}
	if state.Height < 480 {
		state.Height = 800
	}

	return state
}

// Save persists the window state to disk.
func (m *WindowStateManager) Save(state WindowState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}
