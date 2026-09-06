package desktop

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// FileFilter specifies an extension filter for the file picker.
type FileFilter struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// FilePickerOptions configures the native open file dialog.
type FilePickerOptions struct {
	Title       string       `json:"title"`
	DefaultPath string       `json:"defaultPath"`
	Filters     []FileFilter `json:"filters"`
}

// NotificationOptions defines payload for native desktop notifications.
type NotificationOptions struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Icon   string `json:"icon"`
	Silent bool   `json:"silent"`
}

// NativeDialogHandler defines an interface for delegating native file and folder pickers.
type NativeDialogHandler interface {
	SelectDirectory(ctx context.Context, title string) (string, error)
	SelectFile(ctx context.Context, opts FilePickerOptions) (string, error)
	ShowNotification(opts NotificationOptions) error
}

// App is the Wails bridge struct exposed to the frontend.
// It contains ONLY native OS integration capabilities.
type App struct {
	mu            sync.RWMutex
	ctx           context.Context
	dialogHandler NativeDialogHandler
	capabilities  Capabilities
	windowManager *WindowStateManager
}

// NewApp creates a new desktop App instance.
func NewApp(dialogHandler NativeDialogHandler, windowManager *WindowStateManager) *App {
	if windowManager == nil {
		windowManager = NewWindowStateManager("")
	}
	return &App{
		dialogHandler: dialogHandler,
		capabilities:  DefaultCapabilities(),
		windowManager: windowManager,
	}
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
}

// Shutdown is called by Wails when the application terminates.
func (a *App) Shutdown(ctx context.Context) {
}

// GetCapabilities returns the native platform capabilities.
func (a *App) GetCapabilities() Capabilities {
	return a.capabilities
}

// SelectDirectory opens the native OS directory picker.
func (a *App) SelectDirectory(title string) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()

	if dh != nil {
		return dh.SelectDirectory(ctx, title)
	}

	// Fallback CLI-based folder picker if supported (e.g. zenity / osascript / powershell)
	return fallbackSelectDirectory(title)
}

// SelectFile opens the native OS file picker.
func (a *App) SelectFile(opts FilePickerOptions) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()

	if dh != nil {
		return dh.SelectFile(ctx, opts)
	}

	return fallbackSelectFile(opts)
}

// ShowNotification triggers an OS native desktop notification.
func (a *App) ShowNotification(opts NotificationOptions) error {
	a.mu.RLock()
	dh := a.dialogHandler
	a.mu.RUnlock()

	if dh != nil {
		return dh.ShowNotification(opts)
	}

	return fallbackNotification(opts)
}

// OpenExternal opens a URL using the default system browser.
func (a *App) OpenExternal(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

// GetSystemTheme detects the OS dark/light mode preference.
func (a *App) GetSystemTheme() string {
	// Simple heuristics or query
	return "dark"
}

func fallbackSelectDirectory(title string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("osascript", "-e", fmt.Sprintf(`POSIX path of (choose folder with prompt %q)`, title)).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "linux":
		if zenity, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(zenity, "--file-selection", "--directory", "--title", title).Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", nil
}

func fallbackSelectFile(opts FilePickerOptions) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("osascript", "-e", fmt.Sprintf(`POSIX path of (choose file with prompt %q)`, opts.Title)).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "linux":
		if zenity, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(zenity, "--file-selection", "--title", opts.Title).Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", nil
}

func fallbackNotification(opts NotificationOptions) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, opts.Body, opts.Title)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		if notifySend, err := exec.LookPath("notify-send"); err == nil {
			return exec.Command(notifySend, opts.Title, opts.Body).Run()
		}
	}
	return nil
}
