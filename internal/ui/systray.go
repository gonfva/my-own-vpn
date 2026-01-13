package ui

import (
	"sync"

	"github.com/getlantern/systray"
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

	state    ConnectionState
	cost     string
	errorMsg string

	// Menu items
	mStatus     *systray.MenuItem
	mConnect    *systray.MenuItem
	mDisconnect *systray.MenuItem
	mSettings   *systray.MenuItem
	mCost       *systray.MenuItem
	mQuit       *systray.MenuItem

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

// SetCallbacks sets the callback functions for menu actions
func (t *TrayApp) SetCallbacks(onConnect, onDisconnect, onSettings, onQuit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onConnect = onConnect
	t.onDisconnect = onDisconnect
	t.onSettings = onSettings
	t.onQuit = onQuit
}

// Run starts the system tray application (blocks until quit)
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

// onReady is called when the systray is ready
func (t *TrayApp) onReady() {
	// Set initial icon and tooltip
	systray.SetIcon(IconDisconnected())
	systray.SetTitle("My Own VPN")
	systray.SetTooltip("My Own VPN - Disconnected")

	// Create menu items
	t.mStatus = systray.AddMenuItem("Status: Disconnected", "Current connection status")
	t.mStatus.Disable()

	systray.AddSeparator()

	t.mConnect = systray.AddMenuItem("Connect", "Connect to VPN")
	t.mDisconnect = systray.AddMenuItem("Disconnect", "Disconnect from VPN")
	t.mDisconnect.Hide()

	t.mSettings = systray.AddMenuItem("Settings...", "Open settings")

	systray.AddSeparator()

	t.mCost = systray.AddMenuItem("Session Cost: $0.00", "Current session cost")
	t.mCost.Disable()
	t.mCost.Hide()

	systray.AddSeparator()

	t.mQuit = systray.AddMenuItem("Quit", "Exit application")

	// Start menu event handler
	go t.handleMenuEvents()
}

// onExit is called when the systray is exiting
func (t *TrayApp) onExit() {
	// Cleanup if needed
}

// handleMenuEvents handles menu click events
func (t *TrayApp) handleMenuEvents() {
	for {
		select {
		case <-t.mConnect.ClickedCh:
			t.mu.RLock()
			cb := t.onConnect
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		case <-t.mDisconnect.ClickedCh:
			t.mu.RLock()
			cb := t.onDisconnect
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		case <-t.mSettings.ClickedCh:
			t.mu.RLock()
			cb := t.onSettings
			t.mu.RUnlock()
			if cb != nil {
				go cb()
			}
		case <-t.mQuit.ClickedCh:
			t.mu.RLock()
			cb := t.onQuit
			t.mu.RUnlock()
			if cb != nil {
				cb()
			}
			systray.Quit()
			return
		}
	}
}

// UpdateStatus updates the UI to reflect connection status
func (t *TrayApp) UpdateStatus(status string, connected bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if connected {
		t.state = StateConnected
		systray.SetIcon(IconConnected())
		systray.SetTooltip("My Own VPN - Connected")
		t.mConnect.Hide()
		t.mDisconnect.Show()
		t.mCost.Show()
	} else {
		t.state = StateDisconnected
		systray.SetIcon(IconDisconnected())
		systray.SetTooltip("My Own VPN - Disconnected")
		t.mConnect.Show()
		t.mDisconnect.Hide()
		t.mCost.Hide()
	}

	t.mStatus.SetTitle("Status: " + status)
	t.mConnect.Enable()
	t.mDisconnect.Enable()
}

// SetConnecting updates the UI for connecting state
func (t *TrayApp) SetConnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state = StateConnecting
	systray.SetIcon(IconConnecting())
	systray.SetTooltip("My Own VPN - Connecting...")
	t.mStatus.SetTitle("Status: Connecting...")
	t.mConnect.Disable()
	t.mDisconnect.Disable()
}

// SetDisconnecting updates the UI for disconnecting state
func (t *TrayApp) SetDisconnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state = StateDisconnecting
	systray.SetIcon(IconConnecting()) // Use same icon for transitional states
	systray.SetTooltip("My Own VPN - Disconnecting...")
	t.mStatus.SetTitle("Status: Disconnecting...")
	t.mConnect.Disable()
	t.mDisconnect.Disable()
}

// UpdateCost updates the displayed session cost
func (t *TrayApp) UpdateCost(cost string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cost = cost
	if t.mCost != nil {
		t.mCost.SetTitle("Session Cost: " + cost)
	}
}

// SetError updates the UI for error state
func (t *TrayApp) SetError(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state = StateError
	t.errorMsg = message
	systray.SetIcon(IconError())
	systray.SetTooltip("My Own VPN - Error")
	t.mStatus.SetTitle("Status: Error - " + message)
	t.mConnect.Show()
	t.mConnect.Enable()
	t.mDisconnect.Hide()
	t.mCost.Hide()
}

// GetState returns the current connection state
func (t *TrayApp) GetState() ConnectionState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// Quit triggers a clean shutdown of the tray application
func (t *TrayApp) Quit() {
	systray.Quit()
}
