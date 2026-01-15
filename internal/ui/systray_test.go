package ui

import (
	"testing"
)

func TestNewTrayApp(t *testing.T) {
	app := NewTrayApp()

	if app == nil {
		t.Fatal("NewTrayApp returned nil")
	}

	if app.state != StateDisconnected {
		t.Errorf("expected initial state to be StateDisconnected, got %v", app.state)
	}

	if app.cost != "$0.00" {
		t.Errorf("expected initial cost to be '$0.00', got %v", app.cost)
	}
}

func TestSetCallbacks(t *testing.T) {
	app := NewTrayApp()

	connectCalled := false
	disconnectCalled := false
	settingsCalled := false
	quitCalled := false

	app.SetCallbacks(
		func() { connectCalled = true },
		func() { disconnectCalled = true },
		func() { settingsCalled = true },
		func() { quitCalled = true },
	)

	// Verify callbacks are set (by checking they're not nil)
	app.mu.RLock()
	if app.onConnect == nil {
		t.Error("onConnect callback not set")
	}
	if app.onDisconnect == nil {
		t.Error("onDisconnect callback not set")
	}
	if app.onSettings == nil {
		t.Error("onSettings callback not set")
	}
	if app.onQuit == nil {
		t.Error("onQuit callback not set")
	}
	app.mu.RUnlock()

	// Verify callbacks can be called
	app.onConnect()
	app.onDisconnect()
	app.onSettings()
	app.onQuit()

	if !connectCalled {
		t.Error("connect callback was not called")
	}
	if !disconnectCalled {
		t.Error("disconnect callback was not called")
	}
	if !settingsCalled {
		t.Error("settings callback was not called")
	}
	if !quitCalled {
		t.Error("quit callback was not called")
	}
}

func TestGetState(t *testing.T) {
	app := NewTrayApp()

	if app.GetState() != StateDisconnected {
		t.Errorf("expected StateDisconnected, got %v", app.GetState())
	}

	// Test state changes
	app.mu.Lock()
	app.state = StateConnected
	app.mu.Unlock()

	if app.GetState() != StateConnected {
		t.Errorf("expected StateConnected, got %v", app.GetState())
	}
}

func TestConnectionState(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected int
	}{
		{StateDisconnected, 0},
		{StateConnecting, 1},
		{StateConnected, 2},
		{StateDisconnecting, 3},
		{StateError, 4},
	}

	for _, tt := range tests {
		if int(tt.state) != tt.expected {
			t.Errorf("ConnectionState %v has value %d, expected %d", tt.state, tt.state, tt.expected)
		}
	}
}

func TestIconFunctions(t *testing.T) {
	// Test that icon functions return non-empty data
	disconnected := IconDisconnected()
	if len(disconnected) == 0 {
		t.Error("IconDisconnected returned empty data")
	}

	connecting := IconConnecting()
	if len(connecting) == 0 {
		t.Error("IconConnecting returned empty data")
	}

	connected := IconConnected()
	if len(connected) == 0 {
		t.Error("IconConnected returned empty data")
	}

	errorIcon := IconError()
	if len(errorIcon) == 0 {
		t.Error("IconError returned empty data")
	}

	// Verify icons are different from each other
	if string(disconnected) == string(connected) {
		t.Error("disconnected and connected icons should be different")
	}
	if string(connected) == string(errorIcon) {
		t.Error("connected and error icons should be different")
	}
}

func TestSetup(t *testing.T) {
	app := NewTrayApp()

	// Setup should not panic without a Fyne app (graceful no-op)
	app.Setup()

	// State should still be disconnected
	if app.GetState() != StateDisconnected {
		t.Errorf("expected StateDisconnected after Setup, got %v", app.GetState())
	}
}

func TestUpdateMethods(t *testing.T) {
	app := NewTrayApp()

	// These methods should not panic without a Fyne app (graceful no-op)
	app.UpdateStatus("Test", true)
	app.SetConnecting()
	app.SetDisconnecting()
	app.UpdateCost("$1.23")
	app.SetError("test error")
	app.Quit()

	// After SetError, state should be StateError
	if app.GetState() != StateError {
		t.Errorf("expected StateError after SetError, got %v", app.GetState())
	}
}
