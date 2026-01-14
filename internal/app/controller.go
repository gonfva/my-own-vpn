package app

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Controller manages the VPN lifecycle state machine
type Controller struct {
	mu    sync.RWMutex // Thread-safe access to all fields
	state State        // Current state of the state machine

	// Callbacks for UI notifications
	onStateChange func(state State, message string)
	onError       func(err error)

	// Session tracking
	sessionStart time.Time // When connection was established
	errorMsg     string    // Last error message (for Error state)

	// Cancellation support
	cancelFunc context.CancelFunc // For aborting current operation
}

// validTransitions defines allowed state transitions
// Key is current state, value is set of allowed next states
var validTransitions = map[State]map[State]bool{
	StateDisconnected: {
		StateProvisioning: true, // Connect initiated
	},
	StateProvisioning: {
		StateConnecting: true, // Provision successful
		StateError:      true, // Provision failed
	},
	StateConnecting: {
		StateConnected: true, // Connection established
		StateError:     true, // Connection failed
	},
	StateConnected: {
		StateDisconnecting: true, // Disconnect initiated
	},
	StateDisconnecting: {
		StateDeprovisioning: true, // Disconnection completed
	},
	StateDeprovisioning: {
		StateDisconnected: true, // Cleanup completed
	},
	StateError: {
		StateDisconnected: true, // Cleanup after error
	},
}

// NewController creates a new Controller instance
func NewController() *Controller {
	return &Controller{
		state: StateDisconnected,
	}
}

// State returns the current state (thread-safe)
func (c *Controller) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetOnStateChange sets the callback function called when state changes
func (c *Controller) SetOnStateChange(callback func(state State, message string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = callback
}

// SetOnError sets the callback function called when an error occurs
func (c *Controller) SetOnError(callback func(err error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = callback
}

// Connect initiates the connection flow
// This is a stub implementation that simulates the provisioning and connection process
func (c *Controller) Connect(ctx context.Context) error {
	c.mu.Lock()

	// Check if we're already connected or connecting
	if c.state != StateDisconnected {
		c.mu.Unlock()
		return fmt.Errorf("cannot connect: current state is %s", c.state)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	// Transition to Provisioning
	if err := c.setState(StateProvisioning, "Starting provisioning..."); err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()

	// Launch async operation
	go c.doConnect(ctx)

	return nil
}

// Disconnect initiates the disconnection flow
// This is a stub implementation that simulates the disconnection and deprovisioning process
func (c *Controller) Disconnect(ctx context.Context) error {
	c.mu.Lock()

	// Check if we're connected
	if c.state != StateConnected {
		c.mu.Unlock()
		return fmt.Errorf("cannot disconnect: not connected (state: %s)", c.state)
	}

	// Transition to Disconnecting
	if err := c.setState(StateDisconnecting, "Disconnecting..."); err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()

	// Launch async operation
	go c.doDisconnect(ctx)

	return nil
}

// Cancel aborts the current operation
func (c *Controller) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Call cancel function if operation in progress
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
}

// SessionDuration returns the time since connection was established
// Returns 0 if not connected
func (c *Controller) SessionDuration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If not connected, return 0
	if c.state != StateConnected || c.sessionStart.IsZero() {
		return 0
	}

	// Return time elapsed since connection
	return time.Since(c.sessionStart)
}

// canTransition checks if transition from -> to is valid
func (c *Controller) canTransition(from, to State) bool {
	if allowed, ok := validTransitions[from]; ok {
		return allowed[to]
	}
	return false
}

// setState transitions to new state with validation
// Must be called with lock held
func (c *Controller) setState(newState State, message string) error {
	if !c.canTransition(c.state, newState) {
		return fmt.Errorf("invalid state transition: %s -> %s", c.state, newState)
	}

	c.state = newState

	// Special handling for Connected state
	if newState == StateConnected {
		c.sessionStart = time.Now()
	}

	// Special handling for Disconnected state
	if newState == StateDisconnected {
		c.sessionStart = time.Time{} // Zero value
		c.errorMsg = ""
		c.cancelFunc = nil
	}

	// Special handling for Error state
	if newState == StateError && message != "" {
		c.errorMsg = message
	}

	// Notify callback (non-blocking)
	if c.onStateChange != nil {
		go c.onStateChange(newState, message)
	}

	return nil
}

// doConnect performs the connection process (stub implementation)
func (c *Controller) doConnect(ctx context.Context) {
	// STUB: Simulate provisioning
	select {
	case <-ctx.Done():
		c.handleError(ctx.Err(), "Operation cancelled during provisioning")
		return
	case <-time.After(2 * time.Second):
		// Continue
	}

	c.mu.Lock()
	if err := c.setState(StateConnecting, "Establishing connection..."); err != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// STUB: Simulate connecting
	select {
	case <-ctx.Done():
		c.handleError(ctx.Err(), "Operation cancelled during connection")
		return
	case <-time.After(2 * time.Second):
		// Continue
	}

	c.mu.Lock()
	if err := c.setState(StateConnected, "Connected successfully"); err != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
}

// doDisconnect performs the disconnection process (stub implementation)
func (c *Controller) doDisconnect(ctx context.Context) {
	// STUB: Simulate disconnecting
	select {
	case <-ctx.Done():
		// For disconnect, we continue even if cancelled to ensure cleanup
	case <-time.After(1 * time.Second):
		// Continue
	}

	c.mu.Lock()
	if err := c.setState(StateDeprovisioning, "Cleaning up resources..."); err != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// STUB: Simulate deprovisioning
	select {
	case <-ctx.Done():
		// Continue cleanup regardless
	case <-time.After(1 * time.Second):
		// Continue
	}

	c.mu.Lock()
	if err := c.setState(StateDisconnected, "Disconnected"); err != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
}

// handleError is called when an error occurs during operations
func (c *Controller) handleError(err error, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Transition to Error state
	if setErr := c.setState(StateError, message); setErr != nil {
		// If we can't transition, we have a serious problem
		// In production, use proper logger
		return
	}

	// Notify error callback
	if c.onError != nil {
		go c.onError(err)
	}

	// Attempt cleanup in background
	go c.cleanup()
}

// cleanup performs cleanup after error
func (c *Controller) cleanup() {
	// STUB: In future, this will:
	// 1. Call provider.Deprovision()
	// 2. Tear down WireGuard tunnel
	// 3. Release resources

	time.Sleep(500 * time.Millisecond) // Simulate cleanup

	c.mu.Lock()
	defer c.mu.Unlock()

	// Transition back to Disconnected after cleanup
	_ = c.setState(StateDisconnected, "Cleanup completed")
}
