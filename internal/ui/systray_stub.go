//go:build !cgo

package ui

import (
	"sync"
)

// ConnectionState represents the current VPN connection state
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateDisconnecting
	StateError
)

// TrayApp manages the system tray application (stub for non-CGO builds)
type TrayApp struct {
	mu sync.RWMutex

	state ConnectionState
	cost  string

	// Callbacks
	onConnect    func()
	onDisconnect func()
	onSettings   func()
	onQuit       func()
}

// NewTrayApp creates a new TrayApp instance
func NewTrayApp() *TrayApp {
	return &TrayApp{
		state: StateDisconnected,
		cost:  "$0.00",
	}
}

// SetFyneApp sets the Fyne application instance (stub - no-op without CGO)
func (t *TrayApp) SetFyneApp(app interface{}) {
	// No-op in stub implementation
}

// SetCallbacks sets the callback functions for menu actions
func (t *TrayApp) SetCallbacks(onConnect, onDisconnect, onSettings, onQuit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onConnect = onConnect
	t.onDisconnect = onDisconnect
	t.onSettings = onSettings
	t.onQuit = onQuit
}

// Setup initializes the system tray (stub - no-op without CGO)
func (t *TrayApp) Setup() {
	// No-op in stub implementation
}

// UpdateStatus updates the UI to reflect connection status (stub - no-op without CGO)
func (t *TrayApp) UpdateStatus(status string, connected bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if connected {
		t.state = StateConnected
	} else {
		t.state = StateDisconnected
	}
}

// SetConnecting updates the UI for connecting state (stub - no-op without CGO)
func (t *TrayApp) SetConnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateConnecting
}

// SetDisconnecting updates the UI for disconnecting state (stub - no-op without CGO)
func (t *TrayApp) SetDisconnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateDisconnecting
}

// UpdateCost updates the displayed session cost (stub - no-op without CGO)
func (t *TrayApp) UpdateCost(cost string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cost = cost
}

// SetError updates the UI for error state (stub - no-op without CGO)
func (t *TrayApp) SetError(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateError
}

// GetState returns the current connection state
func (t *TrayApp) GetState() ConnectionState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// Quit triggers a clean shutdown of the tray application (stub - no-op without CGO)
func (t *TrayApp) Quit() {
	// No-op in stub implementation
}
