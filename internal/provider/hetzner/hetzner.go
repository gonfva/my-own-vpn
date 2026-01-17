// Package hetzner implements the cloud provider interface for Hetzner Cloud.
package hetzner

import (
	"context"
	"errors"
	"fmt"

	"github.com/gonfva/my-own-vpn/internal/provider"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Common errors returned by the Hetzner provider
var (
	ErrInvalidCredentials = errors.New("invalid Hetzner API token")
	ErrNotProvisioned     = errors.New("no infrastructure provisioned")
)

// Provider implements the provider.Provider interface for Hetzner Cloud.
type Provider struct {
	client *hcloud.Client

	// Session ID for labeling and identifying resources
	sessionID string

	// Track created resources for cleanup
	serverID   int64
	firewallID int64
	sshKeyID   int64

	// SSH key for server access (used to retrieve WireGuard public key)
	sshKey *sshKeyPair

	// Cached server info for GetStatus
	serverPublicKey string
	serverPublicIP  string
}

// New creates a new Hetzner provider with the given API token.
func New(token string) (*Provider, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	return &Provider{
		client: client,
	}, nil
}

// ValidateCredentials verifies that the configured API token is valid
// by calling the Datacenter List API.
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	_, _, err := p.client.Datacenter.List(ctx, hcloud.DatacenterListOpts{})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}
	return nil
}

// ListRegions returns the available Hetzner locations (datacenters).
func (p *Provider) ListRegions(ctx context.Context) ([]provider.Region, error) {
	locations, _, err := p.client.Location.List(ctx, hcloud.LocationListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	regions := make([]provider.Region, 0, len(locations))
	for _, loc := range locations {
		regions = append(regions, provider.Region{
			ID:   loc.Name,
			Name: locationDisplayName(loc.Name, loc.City),
		})
	}

	return regions, nil
}

// Provision creates all necessary Hetzner infrastructure for a VPN server.
// It creates a firewall, SSH key, and server with WireGuard configured via cloud-init.
func (p *Provider) Provision(ctx context.Context, cfg provider.ProvisionConfig) (*provider.ServerInfo, error) {
	// Generate a unique session ID for this provisioning session
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}
	p.sessionID = sessionID

	// Step 1: Create firewall
	if err := p.createFirewall(ctx); err != nil {
		_ = p.deleteFirewall(ctx)
		_ = clearState()
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	// Save state after firewall is created
	if err := p.saveState(); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}

	// Step 2: Create SSH key
	if err := p.createSSHKey(ctx); err != nil {
		_ = p.deleteSSHKey(ctx)
		_ = p.deleteFirewall(ctx)
		_ = clearState()
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	// Save state after SSH key is created
	if err := p.saveState(); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}

	// Step 3: Create server with cloud-init
	if err := p.createServer(ctx, cfg); err != nil {
		_ = p.deleteServer(ctx)
		_ = p.deleteSSHKey(ctx)
		_ = p.deleteFirewall(ctx)
		_ = clearState()
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Save state after server is created
	if err := p.saveState(); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}

	// Step 4: Wait for WireGuard to be ready and get the public key
	pubKey, err := p.waitForWireGuardReady(ctx)
	if err != nil {
		_ = p.deleteServer(ctx)
		_ = p.deleteSSHKey(ctx)
		_ = p.deleteFirewall(ctx)
		_ = clearState()
		return nil, fmt.Errorf("failed waiting for WireGuard: %w", err)
	}
	p.serverPublicKey = pubKey

	// Save final state
	if err := p.saveState(); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}

	return &provider.ServerInfo{
		PublicIP:        p.serverPublicIP,
		WireGuardPort:   wireGuardPort,
		ServerPublicKey: pubKey,
	}, nil
}

// Deprovision tears down all Hetzner infrastructure created by Provision.
func (p *Provider) Deprovision(ctx context.Context) error {
	if p.serverID == 0 && p.firewallID == 0 && p.sshKeyID == 0 {
		return ErrNotProvisioned
	}

	var errs []error

	// Step 1: Delete server (must be done before firewall cleanup)
	if err := p.deleteServer(ctx); err != nil {
		errs = append(errs, fmt.Errorf("delete server: %w", err))
	}

	// Step 2: Delete SSH key
	if err := p.deleteSSHKey(ctx); err != nil {
		errs = append(errs, fmt.Errorf("delete SSH key: %w", err))
	}

	// Step 3: Delete firewall
	if err := p.deleteFirewall(ctx); err != nil {
		errs = append(errs, fmt.Errorf("delete firewall: %w", err))
	}

	// Clear cached server info
	p.serverPublicKey = ""
	p.serverPublicIP = ""
	p.sessionID = ""

	// Clear persisted state
	if err := clearState(); err != nil {
		errs = append(errs, fmt.Errorf("clear state: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to deprovision: %v", errs)
	}

	return nil
}

// GetStatus returns the current status of the provisioned infrastructure.
func (p *Provider) GetStatus(ctx context.Context) (*provider.InfraStatus, error) {
	if p.serverID == 0 {
		return &provider.InfraStatus{
			State:   provider.StateNotFound,
			Message: "No infrastructure provisioned",
		}, nil
	}

	// Get the server
	server, _, err := p.client.Server.GetByID(ctx, p.serverID)
	if err != nil {
		return &provider.InfraStatus{
			State:   provider.StateError,
			Message: fmt.Sprintf("Failed to get server: %v", err),
		}, nil
	}

	if server == nil {
		return &provider.InfraStatus{
			State:   provider.StateNotFound,
			Message: "Server has been deleted",
		}, nil
	}

	// Map Hetzner server status to InfraStatus
	switch server.Status {
	case hcloud.ServerStatusInitializing:
		return &provider.InfraStatus{
			State:   provider.StateProvisioning,
			Message: "Server is initializing",
		}, nil
	case hcloud.ServerStatusStarting:
		return &provider.InfraStatus{
			State:   provider.StateProvisioning,
			Message: "Server is starting",
		}, nil
	case hcloud.ServerStatusRunning:
		return &provider.InfraStatus{
			State:   provider.StateRunning,
			Message: fmt.Sprintf("Server running at %s", p.serverPublicIP),
		}, nil
	case hcloud.ServerStatusStopping:
		return &provider.InfraStatus{
			State:   provider.StateStopped,
			Message: "Server is stopping",
		}, nil
	case hcloud.ServerStatusOff:
		return &provider.InfraStatus{
			State:   provider.StateStopped,
			Message: "Server is stopped",
		}, nil
	default:
		return &provider.InfraStatus{
			State:   provider.StateError,
			Message: fmt.Sprintf("Unknown server status: %s", server.Status),
		}, nil
	}
}

// EstimateCost returns an estimate of the hourly cost for the given config.
func (p *Provider) EstimateCost(cfg provider.ProvisionConfig) provider.CostEstimate {
	// Hetzner pricing (EUR/month, converted to hourly USD approximation)
	// CX11: ~3.29 EUR/month, CX21: ~5.83 EUR/month, CX31: ~10.59 EUR/month
	// Using 730 hours/month and 1.1 EUR/USD conversion rate
	costs := map[string]float64{
		"cx11":  0.0050, // ~3.29 EUR/month
		"cx21":  0.0088, // ~5.83 EUR/month
		"cx31":  0.0159, // ~10.59 EUR/month
		"cpx11": 0.0062, // ~4.15 EUR/month
		"cpx21": 0.0106, // ~7.19 EUR/month
		"cpx31": 0.0182, // ~12.49 EUR/month
		"cax11": 0.0053, // ~3.29 EUR/month (ARM)
		"cax21": 0.0088, // ~5.69 EUR/month (ARM)
		"cax31": 0.0124, // ~8.09 EUR/month (ARM)
	}

	rate, ok := costs[cfg.InstanceType]
	if !ok {
		// Default to cx11 cost if instance type is unknown
		rate = 0.0050
	}

	return provider.CostEstimate{
		HourlyRate: rate,
		Currency:   "USD",
	}
}

// locationDisplayName returns a human-readable name for a Hetzner location.
func locationDisplayName(locationID, city string) string {
	names := map[string]string{
		"fsn1": "Falkenstein, Germany",
		"nbg1": "Nuremberg, Germany",
		"hel1": "Helsinki, Finland",
		"ash":  "Ashburn, USA",
		"hil":  "Hillsboro, USA",
	}

	if name, ok := names[locationID]; ok {
		return name
	}
	if city != "" {
		return city
	}
	return locationID
}

// CleanupOrphaned finds and cleans up any orphaned resources that may have been
// left behind due to a crash or improper shutdown.
func (p *Provider) CleanupOrphaned(ctx context.Context) error {
	// First, try to load state from disk
	loaded, err := p.LoadExistingState()
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}

	// If no state was loaded, try to find resources by labels
	if !loaded {
		if err := p.findResourcesByLabel(ctx); err != nil {
			return fmt.Errorf("failed to find resources by label: %w", err)
		}
	}

	// If no resources found, nothing to clean up
	if !p.HasProvisionedResources() {
		return nil
	}

	// Deprovision found resources
	return p.Deprovision(ctx)
}

// GetSessionID returns the current session ID.
func (p *Provider) GetSessionID() string {
	return p.sessionID
}

// Ensure Provider implements provider.Provider at compile time
var _ provider.Provider = (*Provider)(nil)
