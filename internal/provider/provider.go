// Package provider defines the cloud provider interface and common types
// for provisioning VPN infrastructure.
package provider

import (
	"context"
)

// Provider is the interface that all cloud providers must implement.
// It defines the operations needed to provision and manage VPN infrastructure.
type Provider interface {
	// ValidateCredentials verifies that the configured credentials are valid.
	ValidateCredentials(ctx context.Context) error

	// ListRegions returns the available regions for this provider.
	ListRegions(ctx context.Context) ([]Region, error)

	// Provision creates all necessary infrastructure for a VPN server.
	// This includes networking (VPC, subnet, etc.), security groups, and the VM.
	Provision(ctx context.Context, config ProvisionConfig) (*ServerInfo, error)

	// Deprovision tears down all infrastructure created by Provision.
	// It should clean up all resources to avoid ongoing costs.
	Deprovision(ctx context.Context) error

	// GetStatus returns the current status of the provisioned infrastructure.
	GetStatus(ctx context.Context) (*InfraStatus, error)

	// EstimateCost returns an estimate of the hourly cost for the given config.
	EstimateCost(config ProvisionConfig) CostEstimate
}

// Region represents a geographic region where infrastructure can be deployed.
type Region struct {
	// ID is the provider-specific region identifier (e.g., "us-east-1" for AWS)
	ID string
	// Name is a human-readable name for the region
	Name string
}

// ProvisionConfig contains the configuration for provisioning infrastructure.
type ProvisionConfig struct {
	// Region is the provider-specific region identifier
	Region string
	// InstanceType is the provider-specific instance type (e.g., "t3.micro" for AWS)
	InstanceType string
}

// ServerInfo contains information about a provisioned VPN server.
type ServerInfo struct {
	// PublicIP is the public IP address of the server
	PublicIP string
	// WireGuardPort is the port WireGuard is listening on
	WireGuardPort int
	// ServerPublicKey is the WireGuard public key of the server
	ServerPublicKey string
}

// InfraStatus represents the current state of the infrastructure.
type InfraStatus struct {
	// State is the current state (e.g., "provisioning", "running", "stopped", "error")
	State string
	// Message provides additional details about the state
	Message string
}

// InfraState constants for InfraStatus.State
const (
	StateProvisioning = "provisioning"
	StateRunning      = "running"
	StateStopped      = "stopped"
	StateError        = "error"
	StateNotFound     = "not_found"
)

// CostEstimate provides cost information for running infrastructure.
type CostEstimate struct {
	// HourlyRate is the estimated cost per hour
	HourlyRate float64
	// Currency is the currency code (e.g., "USD")
	Currency string
}
