package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gonfva/my-own-vpn/internal/config"
	"github.com/gonfva/my-own-vpn/internal/provider"
)

// newTestController creates a controller with empty dependencies for basic tests
func newTestController() *Controller {
	return NewController(ControllerDeps{})
}

func TestNewController(t *testing.T) {
	c := newTestController()

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
	c := newTestController()

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
	c := newTestController()

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

func TestSessionDuration(t *testing.T) {
	c := newTestController()

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

func TestInvalidStateOperations(t *testing.T) {
	c := newTestController()
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
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}
	mockCreds := &MockCredentialsManager{}

	c := NewController(ControllerDeps{
		CredentialsManager: mockCreds,
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})
	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

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
	c := newTestController()

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
	c.handleError(context.Background(), testErr, "test error")

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
	c := newTestController()

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

// ============ Real flow tests with mocks ============

func TestConnect_Success(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}
	mockCreds := &MockCredentialsManager{}

	c := NewController(ControllerDeps{
		CredentialsManager: mockCreds,
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	connectedDone := make(chan bool, 1)
	var stateSet = make(map[State]bool)
	var mu sync.Mutex

	c.SetOnStateChange(func(state State, _ string) {
		mu.Lock()
		stateSet[state] = true
		mu.Unlock()
		if state == StateConnected {
			connectedDone <- true
		}
	})

	ctx := context.Background()
	err := c.Connect(ctx)

	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	// Wait for Connected state
	select {
	case <-connectedDone:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for Connected state")
	}

	// Small delay to allow all async state callbacks to complete
	time.Sleep(50 * time.Millisecond)

	// Verify all expected states occurred (order may vary due to async callbacks)
	expectedStates := []State{StateProvisioning, StateConnecting, StateConnected}
	mu.Lock()
	for _, expected := range expectedStates {
		if !stateSet[expected] {
			t.Errorf("expected state %v to be reached", expected)
		}
	}
	mu.Unlock()

	if c.State() != StateConnected {
		t.Errorf("expected final state Connected, got %v", c.State())
	}

	// Verify mock calls
	if !mockProvider.ValidateCalled {
		t.Error("expected provider.ValidateCredentials to be called")
	}
	if !mockProvider.ProvisionCalled {
		t.Error("expected provider.Provision to be called")
	}
	if !mockWG.ConnectCalled {
		t.Error("expected wgClient.Connect to be called")
	}
}

func TestDisconnect_Success(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}
	mockWG.SetConnected(true)

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	// Set up as connected
	c.mu.Lock()
	c.state = StateConnected
	c.sessionStart = time.Now()
	c.activeProvider = mockProvider
	c.mu.Unlock()

	done := make(chan bool, 10)
	var stateChanges []State
	var mu sync.Mutex

	c.SetOnStateChange(func(state State, _ string) {
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
		case <-time.After(10 * time.Second):
			t.Fatalf("timeout waiting for state change %d", i+1)
		}
	}

	expectedStates := []State{StateDisconnecting, StateDeprovisioning, StateDisconnected}
	mu.Lock()
	if len(stateChanges) != len(expectedStates) {
		t.Errorf("expected %d state changes, got %d: %v", len(expectedStates), len(stateChanges), stateChanges)
	}
	mu.Unlock()

	if c.State() != StateDisconnected {
		t.Errorf("expected final state Disconnected, got %v", c.State())
	}

	// Verify mock calls
	if !mockWG.DisconnectCalled {
		t.Error("expected wgClient.Disconnect to be called")
	}
	if !mockProvider.DeprovisionCalled {
		t.Error("expected provider.Deprovision to be called")
	}
}

func TestConnect_NoConfig(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	// No config set
	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	err := c.Connect(ctx)

	if err != nil {
		t.Fatalf("Connect should not return immediate error: %v", err)
	}

	// Wait for error callback
	select {
	case <-errorCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected error callback for missing config")
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error, got %v", c.State())
	}
}

func TestConnect_InvalidConfig(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	// Set invalid config
	c.SetConfig(&config.Config{
		Provider: "invalid-provider",
		Region:   "us-east-1",
	})

	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	_ = c.Connect(ctx)

	// Wait for error callback
	select {
	case <-errorCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected error callback for invalid config")
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error, got %v", c.State())
	}
}

func TestConnect_CredentialValidationFails(t *testing.T) {
	mockProvider := &MockProvider{}
	mockProvider.SetValidateError(errors.New("invalid credentials"))
	mockWG := &MockWireGuardClient{}

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	_ = c.Connect(ctx)

	// Wait for error callback
	select {
	case <-errorCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected error callback for credential validation failure")
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error, got %v", c.State())
	}
}

func TestConnect_ProvisionFails(t *testing.T) {
	mockProvider := &MockProvider{}
	mockProvider.SetProvisionError(errors.New("provision failed"))
	mockWG := &MockWireGuardClient{}

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	_ = c.Connect(ctx)

	// Wait for error callback
	select {
	case <-errorCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected error callback for provision failure")
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Should have attempted cleanup (deprovision)
	if !mockProvider.DeprovisionCalled {
		t.Error("expected provider.Deprovision to be called during cleanup")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error, got %v", c.State())
	}
}

func TestConnect_WireGuardFails(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}
	mockWG.SetConnectError(errors.New("wireguard connection failed"))

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	_ = c.Connect(ctx)

	// Wait for error callback
	select {
	case <-errorCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected error callback for WireGuard failure")
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Should have attempted cleanup (deprovision)
	if !mockProvider.DeprovisionCalled {
		t.Error("expected provider.Deprovision to be called during cleanup")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state after error, got %v", c.State())
	}
}

func TestDisconnect_WireGuardFails(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}
	mockWG.SetConnected(true)
	mockWG.SetDisconnectError(errors.New("wireguard disconnect failed"))

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	// Set up as connected
	c.mu.Lock()
	c.state = StateConnected
	c.sessionStart = time.Now()
	c.activeProvider = mockProvider
	c.mu.Unlock()

	done := make(chan bool, 10)
	c.SetOnStateChange(func(state State, _ string) {
		if state == StateDisconnected {
			done <- true
		}
	})

	ctx := context.Background()
	_ = c.Disconnect(ctx)

	// Should still complete despite WireGuard error
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("expected disconnect to complete despite WireGuard error")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state, got %v", c.State())
	}

	// Deprovision should still be called
	if !mockProvider.DeprovisionCalled {
		t.Error("expected provider.Deprovision to be called even after WireGuard error")
	}
}

func TestDisconnect_DeprovisionFails(t *testing.T) {
	mockProvider := &MockProvider{}
	mockProvider.SetDeprovisionError(errors.New("deprovision failed"))
	mockWG := &MockWireGuardClient{}
	mockWG.SetConnected(true)

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	// Set up as connected
	c.mu.Lock()
	c.state = StateConnected
	c.sessionStart = time.Now()
	c.activeProvider = mockProvider
	c.mu.Unlock()

	done := make(chan bool, 10)
	c.SetOnStateChange(func(state State, _ string) {
		if state == StateDisconnected {
			done <- true
		}
	})

	ctx := context.Background()
	_ = c.Disconnect(ctx)

	// Should still reach Disconnected state despite deprovision error
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("expected disconnect to complete despite deprovision error")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected state, got %v", c.State())
	}
}

func TestConnect_Cancelled(t *testing.T) {
	// Create a slow mock provider that allows us to cancel during provisioning
	slowProvider := &MockProvider{}

	provisionStarted := make(chan bool, 1)
	provisionBlocked := make(chan bool)

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    &MockWireGuardClient{},
		ProviderFactory: func(ctx context.Context, _, _ string) (provider.Provider, error) {
			// Signal that we've entered the provider factory
			select {
			case provisionStarted <- true:
			default:
			}
			// Block until we receive a signal or context is cancelled
			select {
			case <-provisionBlocked:
				return slowProvider, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	errorCalled := make(chan bool, 1)
	c.SetOnError(func(_ error) {
		errorCalled <- true
	})

	ctx := context.Background()
	err := c.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	// Wait for provision to start
	select {
	case <-provisionStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for provisioning to start")
	}

	// Cancel the operation
	c.Cancel()

	// Wait for error callback or timeout
	select {
	case <-errorCalled:
		// Expected - cancellation triggered error
	case <-time.After(2 * time.Second):
		// Also acceptable - if the operation finished before cancel took effect
	}

	// Allow some time for cleanup
	time.Sleep(500 * time.Millisecond)

	// Should be disconnected after cancel and cleanup
	finalState := c.State()
	if finalState != StateError && finalState != StateDisconnected {
		t.Errorf("expected Error or Disconnected state after cancel, got %v", finalState)
	}
}

func TestConnectThenDisconnect(t *testing.T) {
	mockProvider := &MockProvider{}
	mockWG := &MockWireGuardClient{}

	c := NewController(ControllerDeps{
		CredentialsManager: &MockCredentialsManager{},
		WireGuardClient:    mockWG,
		ProviderFactory: func(_ context.Context, _, _ string) (provider.Provider, error) {
			return mockProvider, nil
		},
	})

	c.SetConfig(&config.Config{
		Provider:     "aws",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})

	connectedDone := make(chan bool, 1)
	disconnectedDone := make(chan bool, 1)

	c.SetOnStateChange(func(state State, _ string) {
		switch state {
		case StateConnected:
			connectedDone <- true
		case StateDisconnected:
			disconnectedDone <- true
		}
	})

	ctx := context.Background()

	// Connect
	_ = c.Connect(ctx)

	select {
	case <-connectedDone:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for Connected state")
	}

	if c.State() != StateConnected {
		t.Fatalf("expected Connected, got %v", c.State())
	}

	// Disconnect
	_ = c.Disconnect(ctx)

	select {
	case <-disconnectedDone:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for Disconnected state")
	}

	if c.State() != StateDisconnected {
		t.Errorf("expected Disconnected, got %v", c.State())
	}

	// Verify all operations were called
	if !mockProvider.ValidateCalled {
		t.Error("expected provider.ValidateCredentials to be called")
	}
	if !mockProvider.ProvisionCalled {
		t.Error("expected provider.Provision to be called")
	}
	if !mockWG.ConnectCalled {
		t.Error("expected wgClient.Connect to be called")
	}
	if !mockWG.DisconnectCalled {
		t.Error("expected wgClient.Disconnect to be called")
	}
	if !mockProvider.DeprovisionCalled {
		t.Error("expected provider.Deprovision to be called")
	}
}

func TestSetConfig(t *testing.T) {
	c := newTestController()

	cfg := &config.Config{
		Provider:     "hetzner",
		Region:       "fsn1",
		InstanceType: "cx11",
	}

	c.SetConfig(cfg)

	c.mu.RLock()
	if c.config != cfg {
		t.Error("expected config to be set")
	}
	c.mu.RUnlock()
}
