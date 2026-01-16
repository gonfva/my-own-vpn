package credentials

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Default timeout for validation requests
const defaultValidationTimeout = 30 * time.Second

// Validation error types
var (
	// ErrInvalidCredentials indicates the credentials were rejected by the provider
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrNetworkError indicates a network connectivity problem
	ErrNetworkError = errors.New("network error")
	// ErrValidationTimeout indicates the validation request timed out
	ErrValidationTimeout = errors.New("validation timed out")
)

// Validator tests credentials against cloud provider APIs
type Validator interface {
	// ValidateAWS validates AWS credentials by calling STS GetCallerIdentity
	ValidateAWS(ctx context.Context, creds AWSCredentials) error
	// ValidateHetzner validates Hetzner credentials by listing datacenters
	ValidateHetzner(ctx context.Context, creds HetznerCredentials) error
}

// DefaultValidator implements Validator using real cloud provider APIs
type DefaultValidator struct {
	timeout time.Duration
}

// NewValidator creates a new DefaultValidator with the default timeout
func NewValidator() *DefaultValidator {
	return &DefaultValidator{
		timeout: defaultValidationTimeout,
	}
}

// NewValidatorWithTimeout creates a new DefaultValidator with a custom timeout
func NewValidatorWithTimeout(timeout time.Duration) *DefaultValidator {
	return &DefaultValidator{
		timeout: timeout,
	}
}

// ValidateAWS validates AWS credentials by calling STS GetCallerIdentity.
// This is the recommended way to validate AWS credentials as it works with
// any valid credentials regardless of specific permissions.
func (v *DefaultValidator) ValidateAWS(ctx context.Context, creds AWSCredentials) error {
	if creds.IsEmpty() {
		return fmt.Errorf("%w: credentials are empty", ErrInvalidCredentials)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	// Create AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			"",
		)),
		// Disable shared config to avoid picking up local credentials
		config.WithSharedConfigProfile(""),
	)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Call STS GetCallerIdentity
	client := sts.NewFromConfig(cfg)
	_, err = client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return wrapValidationError(ctx, err, "AWS")
	}

	return nil
}

// ValidateHetzner validates Hetzner credentials by listing datacenters.
// This is a minimal, read-only API call that works with any valid token.
func (v *DefaultValidator) ValidateHetzner(ctx context.Context, creds HetznerCredentials) error {
	if creds.IsEmpty() {
		return fmt.Errorf("%w: credentials are empty", ErrInvalidCredentials)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	// Create Hetzner client
	client := hcloud.NewClient(hcloud.WithToken(creds.APIToken))

	// List datacenters - minimal API call that works with any token
	_, _, err := client.Datacenter.List(ctx, hcloud.DatacenterListOpts{})
	if err != nil {
		return wrapValidationError(ctx, err, "Hetzner")
	}

	return nil
}

// wrapValidationError converts provider-specific errors into user-friendly validation errors
func wrapValidationError(ctx context.Context, err error, provider string) error {
	// Check for context timeout/cancellation
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: could not reach %s API in time", ErrValidationTimeout, provider)
		}
		return fmt.Errorf("%w: %w", ErrNetworkError, ctx.Err())
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return fmt.Errorf("%w: connection to %s timed out", ErrValidationTimeout, provider)
		}
		return fmt.Errorf("%w: could not connect to %s API: %w", ErrNetworkError, provider, err)
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("%w: could not resolve %s API hostname", ErrNetworkError, provider)
	}

	// Default to invalid credentials for authentication failures
	return fmt.Errorf("%w for %s: %w", ErrInvalidCredentials, provider, err)
}

// UserFriendlyError returns a user-friendly error message for validation errors
func UserFriendlyError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return "The credentials are invalid. Please check and try again."
	case errors.Is(err, ErrNetworkError):
		return "Could not connect to the cloud provider. Please check your internet connection."
	case errors.Is(err, ErrValidationTimeout):
		return "Connection to the cloud provider timed out. Please try again."
	default:
		return fmt.Sprintf("Validation failed: %v", err)
	}
}

// STSClient interface for AWS STS operations (used for testing)
type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// Ensure DefaultValidator implements Validator
var _ Validator = (*DefaultValidator)(nil)
