package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gonfva/my-own-vpn/internal/config"
	"github.com/gonfva/my-own-vpn/internal/credentials"
	"github.com/gonfva/my-own-vpn/internal/provider"
	"github.com/gonfva/my-own-vpn/internal/wireguard"
)

// ProviderFactory creates a provider instance for the given type and region
type ProviderFactory func(ctx context.Context, providerType, region string) (provider.Provider, error)

// ControllerDeps contains dependencies for the Controller
type ControllerDeps struct {
	CredentialsManager credentials.Manager
	WireGuardClient    wireguard.Client
	ProviderFactory    ProviderFactory
}

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

	// Dependencies (injected)
	credsMgr        credentials.Manager
	wgClient        wireguard.Client
	providerFactory ProviderFactory

	// Configuration
	config *config.Config

	// Active session state (populated during connect)
	activeProvider provider.Provider
	serverInfo     *provider.ServerInfo
	clientKeyPair  *wireguard.KeyPair
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

// NewController creates a new Controller instance with the given dependencies
func NewController(deps ControllerDeps) *Controller {
	return &Controller{
		state:           StateDisconnected,
		credsMgr:        deps.CredentialsManager,
		wgClient:        deps.WireGuardClient,
		providerFactory: deps.ProviderFactory,
	}
}

// SetConfig sets the configuration for the controller
func (c *Controller) SetConfig(cfg *config.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cfg
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

// doConnect performs the connection process
func (c *Controller) doConnect(ctx context.Context) {
	// Step 1: Validate configuration
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	if cfg == nil {
		c.handleError(ctx, fmt.Errorf("no configuration set"), "Configuration required")
		return
	}

	if err := config.Validate(cfg); err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Invalid configuration: %v", err))
		return
	}

	// Check for cancellation
	if ctx.Err() != nil {
		c.handleError(ctx, ctx.Err(), "Operation cancelled")
		return
	}

	// Step 2: Create provider via factory function
	if c.providerFactory == nil {
		c.handleError(ctx, fmt.Errorf("no provider factory configured"), "Provider factory required")
		return
	}

	prov, err := c.providerFactory(ctx, cfg.Provider, cfg.Region)
	if err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Failed to create provider: %v", err))
		return
	}

	c.mu.Lock()
	c.activeProvider = prov
	c.mu.Unlock()

	// Step 3: Validate credentials
	if err := prov.ValidateCredentials(ctx); err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Invalid credentials: %v", err))
		return
	}

	// Check for cancellation
	if ctx.Err() != nil {
		c.handleError(ctx, ctx.Err(), "Operation cancelled")
		return
	}

	// Step 4: Generate WireGuard key pair
	keyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Failed to generate key pair: %v", err))
		return
	}

	c.mu.Lock()
	c.clientKeyPair = keyPair
	c.mu.Unlock()

	// Step 5: Provision infrastructure (already in Provisioning state)
	serverInfo, err := prov.Provision(ctx, provider.ProvisionConfig{
		Region:       cfg.Region,
		InstanceType: cfg.InstanceType,
	})
	if err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Failed to provision infrastructure: %v", err))
		return
	}

	c.mu.Lock()
	c.serverInfo = serverInfo
	c.mu.Unlock()

	// Check for cancellation
	if ctx.Err() != nil {
		c.handleError(ctx, ctx.Err(), "Operation cancelled during provisioning")
		return
	}

	// Step 6: Transition to Connecting
	c.mu.Lock()
	if err := c.setState(StateConnecting, "Establishing VPN connection..."); err != nil {
		c.mu.Unlock()
		c.handleError(ctx, err, "Failed to transition to connecting state")
		return
	}
	c.mu.Unlock()

	// Step 7: Connect WireGuard
	if c.wgClient == nil {
		c.handleError(ctx, fmt.Errorf("no WireGuard client configured"), "WireGuard client required")
		return
	}

	if err := c.wgClient.Connect(ctx, serverInfo, keyPair); err != nil {
		c.handleError(ctx, err, fmt.Sprintf("Failed to connect WireGuard: %v", err))
		return
	}

	// Step 8: Transition to Connected
	c.mu.Lock()
	if err := c.setState(StateConnected, "Connected successfully"); err != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
}

// doDisconnect performs the disconnection process
func (c *Controller) doDisconnect(ctx context.Context) {
	// Step 1: Disconnect WireGuard (log errors but continue)
	if c.wgClient != nil {
		if err := c.wgClient.Disconnect(ctx); err != nil {
			log.Printf("Warning: failed to disconnect WireGuard: %v", err)
		}
	}

	// Step 2: Transition to Deprovisioning
	c.mu.Lock()
	if err := c.setState(StateDeprovisioning, "Cleaning up resources..."); err != nil {
		c.mu.Unlock()
		log.Printf("Warning: failed to transition to deprovisioning: %v", err)
		return
	}
	activeProvider := c.activeProvider
	c.mu.Unlock()

	// Step 3: Deprovision infrastructure (log errors but continue)
	if activeProvider != nil {
		if err := activeProvider.Deprovision(ctx); err != nil {
			log.Printf("Warning: failed to deprovision infrastructure: %v", err)
		}
	}

	// Step 4: Clear session state
	c.mu.Lock()
	c.activeProvider = nil
	c.serverInfo = nil
	c.clientKeyPair = nil
	c.mu.Unlock()

	// Step 5: Transition to Disconnected
	c.mu.Lock()
	if err := c.setState(StateDisconnected, "Disconnected"); err != nil {
		c.mu.Unlock()
		log.Printf("Warning: failed to transition to disconnected: %v", err)
		return
	}
	c.mu.Unlock()
}

// handleError is called when an error occurs during operations
func (c *Controller) handleError(ctx context.Context, err error, message string) {
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
	// Note: We pass the original context but cleanup may use its own
	// context if needed to ensure resources are freed
	go c.cleanup(ctx)
}

// cleanup performs cleanup after error
// Note: We intentionally use a fresh context for cleanup operations
// to ensure they complete even if the original context was cancelled.
// This is critical to avoid leaving cloud resources running.
//
//nolint:contextcheck // Intentionally using fresh context for cleanup
func (c *Controller) cleanup(_ context.Context) {
	// Use a fresh context for cleanup to ensure it completes
	// even if the original context was cancelled
	cleanupCtx := context.Background()

	// Step 1: Check if WireGuard is connected and disconnect if so
	if c.wgClient != nil {
		status := c.wgClient.Status()
		if status.Connected {
			if err := c.wgClient.Disconnect(cleanupCtx); err != nil {
				log.Printf("Warning: failed to disconnect WireGuard during cleanup: %v", err)
			}
		}
	}

	// Step 2: Check if provider exists and deprovision if so
	c.mu.Lock()
	activeProvider := c.activeProvider
	c.mu.Unlock()

	if activeProvider != nil {
		if err := activeProvider.Deprovision(cleanupCtx); err != nil {
			log.Printf("Warning: failed to deprovision during cleanup: %v", err)
		}
	}

	// Step 3: Clear session state
	c.mu.Lock()
	c.activeProvider = nil
	c.serverInfo = nil
	c.clientKeyPair = nil
	c.mu.Unlock()

	// Step 4: Transition back to Disconnected after cleanup
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.setState(StateDisconnected, "Cleanup completed")
}
