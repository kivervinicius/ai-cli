package desktop

// Capabilities represents the native platform features supported by Nexus Desktop.
type Capabilities struct {
	Native           bool `json:"native"`
	FilePicker       bool `json:"filePicker"`
	FolderPicker     bool `json:"folderPicker"`
	Notifications    bool `json:"notifications"`
	Tray             bool `json:"tray"`
	NativeMenus      bool `json:"nativeMenus"`
	DeepLinks        bool `json:"deepLinks"`
	AutoStart        bool `json:"autoStart"`
	WindowManagement bool `json:"windowManagement"`
}

// DefaultCapabilities returns the standard native desktop capabilities.
func DefaultCapabilities() Capabilities {
	return Capabilities{
		Native:           true,
		FilePicker:       true,
		FolderPicker:     true,
		Notifications:    true,
		Tray:             true,
		NativeMenus:      true,
		DeepLinks:        true,
		AutoStart:        true,
		WindowManagement: true,
	}
}
