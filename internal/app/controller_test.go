package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewController(t *testing.T) {
	c := NewController()

	if c == nil {
		t.Fatal("NewController returned nil")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected initial state StateDisconnected, got %v", c.State())
	}

	if c.SessionDuration() != 0 {
		t.Errorf("expected session duration 0, got %v", c.SessionDuration())
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateDisconnected, "Disconnected"},
		{StateProvisioning, "Provisioning"},
		{StateConnecting, "Connecting"},
		{StateConnected, "Connected"},
		{StateDisconnecting, "Disconnecting"},
		{StateDeprovisioning, "Deprovisioning"},
		{StateError, "Error"},
		{State(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

func TestValidTransitions(t *testing.T) {
	c := NewController()

	tests := []struct {
		from  State
		to    State
		valid bool
	}{
		{StateDisconnected, StateProvisioning, true},
		{StateDisconnected, StateConnected, false},
		{StateDisconnected, StateConnecting, false},
		{StateProvisioning, StateConnecting, true},
		{StateProvisioning, StateError, true},
		{StateProvisioning, StateDisconnected, false},
		{StateConnecting, StateConnected, true},
		{StateConnecting, StateError, true},
		{StateConnecting, StateDisconnecting, false},
		{StateConnected, StateDisconnecting, true},
		{StateConnected, StateDisconnected, false},
		{StateConnected, StateProvisioning, false},
		{StateDisconnecting, StateDeprovisioning, true},
		{StateDisconnecting, StateDisconnected, false},
		{StateDeprovisioning, StateDisconnected, true},
		{StateDeprovisioning, StateError, false},
		{StateError, StateDisconnected, true},
		{StateError, StateConnecting, false},
	}

	for _, tt := range tests {
		got := c.canTransition(tt.from, tt.to)
		if got != tt.valid {
			t.Errorf("canTransition(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.valid)
		}
	}
}

func TestSetStateCallbacks(t *testing.T) {
	c := NewController()

	done := make(chan bool)
	var receivedState State
	var receivedMsg string

	c.SetOnStateChange(func(state State, msg string) {
		receivedState = state
		receivedMsg = msg
		done <- true
	})

	// Manually set state to test callback
	c.mu.Lock()
	err := c.setState(StateProvisioning, "test message")
	c.mu.Unlock()

	if err != nil {
		t.Fatalf("setState returned error: %v", err)
	}

	// Wait for callback
	select {
	case <-done:
		// Callback was called
	case <-time.After(1 * time.Second):
		t.Fatal("state change callback not called within timeout")
	}

	if receivedState != StateProvisioning {
		t.Errorf("expected StateProvisioning, got %v", receivedState)
	}

	if receivedMsg != "test message" {
		t.Errorf("expected 'test message', got %q", receivedMsg)
	}
}

func TestConnectStub(t *testing.T) {
	c := NewController()

	done := make(chan bool, 10)
	var stateChanges []State
	var mu sync.Mutex

	c.SetOnStateChange(func(state State, msg string) {
		mu.Lock()
		stateChanges = append(stateChanges, state)
		mu.Unlock()
		done <- true
	})

	ctx := context.Background()
	err := c.Connect(ctx)

	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	// Wait for 3 state changes: Provisioning -> Connecting -> Connected
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Got state change
		case <-time.After(10 * time.Second):
			t.Fatalf("timeout waiting for state change %d", i+1)
		}
	}

	// Should have transitioned: Provisioning -> Connecting -> Connected
	expectedStates := []State{StateProvisioning, StateConnecting, StateConnected}

	mu.Lock()
	if len(stateChanges) != len(expectedStates) {
		t.Errorf("expected %d state changes, got %d: %v", len(expectedStates), len(stateChanges), stateChanges)
	}

	for i, expected := range expectedStates {
		if i >= len(stateChanges) {
			break
		}
		if stateChanges[i] != expected {
			t.Errorf("state change %d: expected %v, got %v", i, expected, stateChanges[i])
		}
	}
	mu.Unlock()

	if c.State() != StateConnected {
		t.Errorf("expected final state Connected, got %v", c.State())
	}
}

func TestDisconnectStub(t *testing.T) {
	c := NewController()

	// First manually set to Connected state
	c.mu.Lock()
	c.state = StateConnected
	c.sessionStart = time.Now()
	c.mu.Unlock()

	done := make(chan bool, 10)
	var stateChanges []State
	var mu sync.Mutex

	c.SetOnStateChange(func(state State, msg string) {
		mu.Lock()
		stateChanges = append(stateChanges, state)
		mu.Unlock()
		done <- true
	})

	ctx := context.Background()
	err := c.Disconnect(ctx)

	if err != nil {
		t.Fatalf("Disconnect returned error: %v", err)
	}

	// Wait for 3 state changes: Disconnecting -> Deprovisioning -> Disconnected
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Got state change
		case <-time.After(10 * time.Second):
			t.Fatalf("timeout waiting for state change %d", i+1)
		}
	}

	// Should have transitioned: Disconnecting -> Deprovisioning -> Disconnected
	expectedStates := []State{StateDisconnecting, StateDeprovisioning, StateDisconnected}

	mu.Lock()
	if len(stateChanges) != len(expectedStates) {
		t.Errorf("expected %d state changes, got %d: %v", len(expectedStates), len(stateChanges), stateChanges)
	}
	mu.Unlock()

	if c.State() != StateDisconnected {
		t.Errorf("expected final state Disconnected, got %v", c.State())
	}
}

func TestSessionDuration(t *testing.T) {
	c := NewController()

	// When disconnected, duration should be 0
	if d := c.SessionDuration(); d != 0 {
		t.Errorf("expected 0 duration when disconnected, got %v", d)
	}

	// Simulate connection
	c.mu.Lock()
	c.sessionStart = time.Now().Add(-5 * time.Minute)
	c.state = StateConnected
	c.mu.Unlock()

	// Duration should be approximately 5 minutes
	d := c.SessionDuration()
	if d < 4*time.Minute || d > 6*time.Minute {
		t.Errorf("expected ~5 minutes, got %v", d)
	}

	// Manually reset to disconnected state (bypass validation for testing)
	c.mu.Lock()
	c.state = StateDisconnected
	c.sessionStart = time.Time{}
	c.mu.Unlock()

	if d := c.SessionDuration(); d != 0 {
		t.Errorf("expected 0 duration after disconnect, got %v", d)
	}
}

func TestCancel(t *testing.T) {
	c := NewController()

	ctx := context.Background()

	// Start connection
	err := c.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Cancel should work
	c.Cancel()

	// Check that cancelFunc was called
	c.mu.RLock()
	if c.cancelFunc != nil {
		t.Error("cancelFunc should be nil after Cancel()")
	}
	c.mu.RUnlock()

	// Wait for error handling to complete
	time.Sleep(1 * time.Second)

	// Should end up in Error or Disconnected state after cleanup
	finalState := c.State()
	if finalState != StateError && finalState != StateDisconnected {
		t.Errorf("expected Error or Disconnected state after cancel, got %v", finalState)
	}
}

func TestInvalidStateOperations(t *testing.T) {
	c := NewController()
	ctx := context.Background()

	// Try to disconnect when not connected
	err := c.Disconnect(ctx)
	if err == nil {
		t.Error("expected error when disconnecting from Disconnected state")
	}

	// Manually set to Connected
	c.mu.Lock()
	c.state = StateConnected
	c.mu.Unlock()

	// Try to connect when already connected
	err = c.Connect(ctx)
	if err == nil {
		t.Error("expected error when connecting from Connected state")
	}

	// Manually set to Connecting
	c.mu.Lock()
	c.state = StateConnecting
	c.mu.Unlock()

	// Try to connect when in intermediate state
	err = c.Connect(ctx)
	if err == nil {
		t.Error("expected error when connecting from Connecting state")
	}
}

func TestThreadSafety(t *testing.T) {
	c := NewController()

	// Run with -race flag to detect races
	done := make(chan bool)

	// Multiple goroutines reading state
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = c.State()
				_ = c.SessionDuration()
			}
			done <- true
		}()
	}

	// One goroutine writing state
	go func() {
		ctx := context.Background()
		_ = c.Connect(ctx)
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}
}

func TestErrorCallback(t *testing.T) {
	c := NewController()

	// Set controller to a state that can transition to Error (Provisioning)
	c.mu.Lock()
	c.state = StateProvisioning
	c.mu.Unlock()

	errorDone := make(chan bool)
	var receivedError error

	c.SetOnError(func(err error) {
		receivedError = err
		errorDone <- true
	})

	// Manually trigger an error
	testErr := context.Canceled
	c.handleError(testErr, "test error")

	// Wait for error callback
	select {
	case <-errorDone:
		// Error callback was called
	case <-time.After(1 * time.Second):
		t.Fatal("error callback was not called within timeout")
	}

	if !errors.Is(receivedError, testErr) {
		t.Errorf("expected error %v, got %v", testErr, receivedError)
	}

	// Wait for cleanup to complete
	time.Sleep(1 * time.Second)

	// Should be back to Disconnected after cleanup
	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error cleanup, got %v", c.State())
	}
}

func TestInvalidTransitionRejected(t *testing.T) {
	c := NewController()

	// Try an invalid transition
	c.mu.Lock()
	err := c.setState(StateConnected, "invalid")
	c.mu.Unlock()

	if err == nil {
		t.Error("expected error for invalid state transition Disconnected -> Connected")
	}

	// State should remain unchanged
	if c.State() != StateDisconnected {
		t.Errorf("expected state to remain Disconnected, got %v", c.State())
	}
}
