//go:build cgo

package ui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

// TrayApp manages the system tray application
type TrayApp struct {
	mu sync.RWMutex

	fyneApp fyne.App
	state   ConnectionState
	cost    string

	// Menu items
	mStatus     *fyne.MenuItem
	mConnect    *fyne.MenuItem
	mDisconnect *fyne.MenuItem
	mSettings   *fyne.MenuItem
	mCost       *fyne.MenuItem

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

// SetFyneApp sets the Fyne application instance
func (t *TrayApp) SetFyneApp(app fyne.App) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fyneApp = app
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

// Setup initializes the system tray menu and icon
func (t *TrayApp) Setup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.fyneApp == nil {
		return
	}

	// Check if the app supports desktop features (system tray)
	if desk, ok := t.fyneApp.(desktop.App); ok {
		// Set the system tray icon
		desk.SetSystemTrayIcon(IconDisconnectedResource())

		// Create menu items
		t.mStatus = fyne.NewMenuItem("Status: Disconnected", nil)
		t.mStatus.Disabled = true

		t.mConnect = fyne.NewMenuItem("Connect", func() {
			t.mu.RLock()
			cb := t.onConnect
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		})

		t.mDisconnect = fyne.NewMenuItem("Disconnect", func() {
			t.mu.RLock()
			cb := t.onDisconnect
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		})

		t.mSettings = fyne.NewMenuItem("Settings...", func() {
			t.mu.RLock()
			cb := t.onSettings
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		})

		t.mCost = fyne.NewMenuItem("Session Cost: $0.00", nil)
		t.mCost.Disabled = true

		quitItem := fyne.NewMenuItem("Quit", func() {
			t.mu.RLock()
			cb := t.onQuit
			t.mu.RUnlock()
			if cb != nil {
				cb()
			}
			t.fyneApp.Quit()
		})

		// Build the menu
		menu := fyne.NewMenu("My Own VPN",
			t.mStatus,
			fyne.NewMenuItemSeparator(),
			t.mConnect,
			t.mDisconnect,
			t.mSettings,
			fyne.NewMenuItemSeparator(),
			t.mCost,
			fyne.NewMenuItemSeparator(),
			quitItem,
		)

		// Set initial visibility
		t.mDisconnect.Disabled = true
		t.mCost.Disabled = true

		desk.SetSystemTrayMenu(menu)
	}
}

// UpdateStatus updates the UI to reflect connection status
func (t *TrayApp) UpdateStatus(status string, connected bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always update state
	if connected {
		t.state = StateConnected
	} else {
		t.state = StateDisconnected
	}

	// Update UI only if Fyne app is available
	if t.fyneApp == nil {
		return
	}

	desk, ok := t.fyneApp.(desktop.App)
	if !ok {
		return
	}

	if connected {
		desk.SetSystemTrayIcon(IconConnectedResource())
		t.mConnect.Disabled = true
		t.mDisconnect.Disabled = false
		t.mCost.Disabled = false
	} else {
		desk.SetSystemTrayIcon(IconDisconnectedResource())
		t.mConnect.Disabled = false
		t.mDisconnect.Disabled = true
		t.mCost.Disabled = true
	}

	t.mStatus.Label = "Status: " + status
}

// SetConnecting updates the UI for connecting state
func (t *TrayApp) SetConnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always update state
	t.state = StateConnecting

	// Update UI only if Fyne app is available
	if t.fyneApp == nil {
		return
	}

	desk, ok := t.fyneApp.(desktop.App)
	if !ok {
		return
	}

	desk.SetSystemTrayIcon(IconConnectingResource())
	t.mStatus.Label = "Status: Connecting..."
	t.mConnect.Disabled = true
	t.mDisconnect.Disabled = true
}

// SetDisconnecting updates the UI for disconnecting state
func (t *TrayApp) SetDisconnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always update state
	t.state = StateDisconnecting

	// Update UI only if Fyne app is available
	if t.fyneApp == nil {
		return
	}

	desk, ok := t.fyneApp.(desktop.App)
	if !ok {
		return
	}

	desk.SetSystemTrayIcon(IconConnectingResource()) // Use same icon for transitional states
	t.mStatus.Label = "Status: Disconnecting..."
	t.mConnect.Disabled = true
	t.mDisconnect.Disabled = true
}

// UpdateCost updates the displayed session cost
func (t *TrayApp) UpdateCost(cost string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cost = cost
	if t.mCost != nil {
		t.mCost.Label = "Session Cost: " + cost
	}
}

// SetError updates the UI for error state
func (t *TrayApp) SetError(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always update state
	t.state = StateError

	// Update UI only if Fyne app is available
	if t.fyneApp == nil {
		return
	}

	desk, ok := t.fyneApp.(desktop.App)
	if !ok {
		return
	}

	desk.SetSystemTrayIcon(IconErrorResource())
	t.mStatus.Label = "Status: Error - " + message
	t.mConnect.Disabled = false
	t.mDisconnect.Disabled = true
	t.mCost.Disabled = true
}

// GetState returns the current connection state
func (t *TrayApp) GetState() ConnectionState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// Quit triggers a clean shutdown of the tray application
func (t *TrayApp) Quit() {
	t.mu.RLock()
	app := t.fyneApp
	t.mu.RUnlock()

	if app != nil {
		app.Quit()
	}
}
