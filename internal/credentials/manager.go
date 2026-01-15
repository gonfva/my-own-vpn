package credentials

import "context"

// Service name constants for keychain/credential storage
const (
	ServiceName    = "MyOwnVPN"
	AWSAccount     = "aws-credentials"
	HetznerAccount = "hetzner-credentials"
)

// Provider constants
const (
	ProviderAWS     = "aws"
	ProviderHetzner = "hetzner"
)

// Manager defines the interface for credential storage
type Manager interface {
	// SaveAWS stores AWS credentials securely
	SaveAWS(ctx context.Context, creds AWSCredentials) error

	// LoadAWS retrieves stored AWS credentials
	LoadAWS(ctx context.Context) (*AWSCredentials, error)

	// DeleteAWS removes stored AWS credentials
	DeleteAWS(ctx context.Context) error

	// SaveHetzner stores Hetzner credentials securely
	SaveHetzner(ctx context.Context, creds HetznerCredentials) error

	// LoadHetzner retrieves stored Hetzner credentials
	LoadHetzner(ctx context.Context) (*HetznerCredentials, error)

	// DeleteHetzner removes stored Hetzner credentials
	DeleteHetzner(ctx context.Context) error

	// HasCredentials checks if credentials are stored for the given provider
	HasCredentials(ctx context.Context, provider string) bool
}
