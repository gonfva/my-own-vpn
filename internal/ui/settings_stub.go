//go:build !cgo

package ui

import "sync"

// SettingsWindow manages the settings window UI (stub for non-CGO builds)
type SettingsWindow struct {
	mu            sync.Mutex
	visible       bool
	currentConfig SettingsConfig
	onSave        func(config SettingsConfig)
}

// NewSettingsWindow creates a new SettingsWindow instance
func NewSettingsWindow() *SettingsWindow {
	return &SettingsWindow{
		currentConfig: DefaultSettingsConfig(),
	}
}

// SetFyneApp sets the Fyne application instance (stub - no-op without CGO)
func (s *SettingsWindow) SetFyneApp(fyneApp interface{}) {
	// No-op in stub implementation
}

// GetFyneApp returns nil in stub implementation (no Fyne app without CGO)
func (s *SettingsWindow) GetFyneApp() interface{} {
	return nil
}

// Show displays the settings window (stub - no-op without CGO)
func (s *SettingsWindow) Show() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visible = true
}

// Hide hides the settings window (stub - no-op without CGO)
func (s *SettingsWindow) Hide() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visible = false
}

// SetOnSave sets the callback function called when settings are saved
func (s *SettingsWindow) SetOnSave(callback func(config SettingsConfig)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSave = callback
}

// LoadConfig loads configuration into the settings window
func (s *SettingsWindow) LoadConfig(config SettingsConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentConfig = config
}

// GetConfig returns the current configuration
func (s *SettingsWindow) GetConfig() SettingsConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentConfig
}

// IsVisible returns whether the settings window is currently visible
func (s *SettingsWindow) IsVisible() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visible
}

// RunFyneLoop starts the Fyne event loop (stub - no-op without CGO)
func (s *SettingsWindow) RunFyneLoop() {
	// No-op in stub implementation
}

// StopFyneLoop stops the Fyne event loop (stub - no-op without CGO)
func (s *SettingsWindow) StopFyneLoop() {
	// No-op in stub implementation
}
