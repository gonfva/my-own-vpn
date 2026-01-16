// Package aws implements the cloud provider interface for Amazon Web Services.
package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gonfva/my-own-vpn/internal/provider"
)

// Common errors returned by the AWS provider
var (
	ErrInvalidCredentials = errors.New("invalid AWS credentials")
	ErrNotProvisioned     = errors.New("no infrastructure provisioned")
	ErrNotImplemented     = errors.New("not implemented")
)

// Provider implements the provider.Provider interface for AWS.
type Provider struct {
	cfg       aws.Config
	ec2Client *ec2.Client
	stsClient *sts.Client
	region    string

	// Track created resources for cleanup (will be populated by Provision)
	vpcID             string
	internetGatewayID string
	subnetID          string
	routeTableID      string
	securityGroupID   string
	instanceID        string
	keyPairName       string

	// Cached server info for GetStatus
	serverPublicKey string
	serverPublicIP  string
}

// New creates a new AWS provider with the given credentials and region.
// If region is empty, it defaults to "us-east-1".
func New(ctx context.Context, accessKey, secretKey, region string) (*Provider, error) {
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"", // session token (empty for long-term credentials)
		)),
		// Disable shared config to avoid picking up local credentials
		awsconfig.WithSharedConfigProfile(""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Provider{
		cfg:       cfg,
		ec2Client: ec2.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
		region:    region,
	}, nil
}

// ValidateCredentials verifies that the configured AWS credentials are valid
// by calling the STS GetCallerIdentity API.
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	_, err := p.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}
	return nil
}

// ListRegions returns the available AWS regions.
func (p *Provider) ListRegions(ctx context.Context) ([]provider.Region, error) {
	output, err := p.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only include opted-in regions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	regions := make([]provider.Region, 0, len(output.Regions))
	for _, r := range output.Regions {
		if r.RegionName != nil {
			regions = append(regions, provider.Region{
				ID:   *r.RegionName,
				Name: regionDisplayName(*r.RegionName),
			})
		}
	}

	return regions, nil
}

// Provision creates all necessary AWS infrastructure for a VPN server.
// It creates networking resources, launches an EC2 instance with WireGuard,
// and waits for the server to be ready.
func (p *Provider) Provision(ctx context.Context, cfg provider.ProvisionConfig) (*provider.ServerInfo, error) {
	// Step 1: Create networking infrastructure (VPC, IGW, Subnet, Route Table, Security Group)
	if err := p.createNetworking(ctx); err != nil {
		// Attempt to clean up any partially created resources
		_ = p.deleteNetworking(ctx)
		return nil, fmt.Errorf("failed to create networking: %w", err)
	}

	// Step 2: Launch EC2 instance with WireGuard user-data
	if err := p.launchInstance(ctx, cfg.InstanceType); err != nil {
		// Clean up on failure
		_ = p.deleteKeyPair(ctx)
		_ = p.deleteNetworking(ctx)
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	// Step 3: Wait for instance to be running
	if err := p.waitForInstanceRunning(ctx); err != nil {
		// Clean up on failure
		_ = p.terminateInstance(ctx)
		_ = p.deleteKeyPair(ctx)
		_ = p.deleteNetworking(ctx)
		return nil, fmt.Errorf("failed waiting for instance: %w", err)
	}

	// Step 4: Get the public IP
	publicIP, err := p.getInstancePublicIP(ctx)
	if err != nil {
		// Clean up on failure
		_ = p.terminateInstance(ctx)
		_ = p.deleteKeyPair(ctx)
		_ = p.deleteNetworking(ctx)
		return nil, fmt.Errorf("failed to get public IP: %w", err)
	}
	p.serverPublicIP = publicIP

	// Step 5: Wait for WireGuard to be ready and get the public key
	pubKey, err := p.waitForWireGuardReady(ctx)
	if err != nil {
		// Clean up on failure
		_ = p.terminateInstance(ctx)
		_ = p.deleteKeyPair(ctx)
		_ = p.deleteNetworking(ctx)
		return nil, fmt.Errorf("failed waiting for WireGuard: %w", err)
	}
	p.serverPublicKey = pubKey

	return &provider.ServerInfo{
		PublicIP:        publicIP,
		WireGuardPort:   wireGuardPort,
		ServerPublicKey: pubKey,
	}, nil
}

// Deprovision tears down all AWS infrastructure created by Provision.
// It terminates the EC2 instance, deletes the key pair, and cleans up all networking resources.
func (p *Provider) Deprovision(ctx context.Context) error {
	if p.instanceID == "" && p.vpcID == "" {
		return ErrNotProvisioned
	}

	var errs []error

	// Step 1: Terminate EC2 instance (must be done before networking cleanup)
	if err := p.terminateInstance(ctx); err != nil {
		errs = append(errs, fmt.Errorf("terminate instance: %w", err))
	}

	// Step 2: Delete key pair
	if err := p.deleteKeyPair(ctx); err != nil {
		errs = append(errs, fmt.Errorf("delete key pair: %w", err))
	}

	// Step 3: Delete networking resources
	if err := p.deleteNetworking(ctx); err != nil {
		errs = append(errs, fmt.Errorf("delete networking: %w", err))
	}

	// Clear cached server info
	p.serverPublicKey = ""
	p.serverPublicIP = ""

	if len(errs) > 0 {
		return fmt.Errorf("failed to deprovision: %v", errs)
	}

	return nil
}

// GetStatus returns the current status of the provisioned infrastructure.
func (p *Provider) GetStatus(ctx context.Context) (*provider.InfraStatus, error) {
	if p.instanceID == "" {
		return &provider.InfraStatus{
			State:   provider.StateNotFound,
			Message: "No infrastructure provisioned",
		}, nil
	}

	// Get the instance state
	state, err := p.getInstanceState(ctx)
	if err != nil {
		return &provider.InfraStatus{
			State:   provider.StateError,
			Message: fmt.Sprintf("Failed to get instance state: %v", err),
		}, nil
	}

	// Map EC2 instance state to InfraStatus
	switch state {
	case "pending":
		return &provider.InfraStatus{
			State:   provider.StateProvisioning,
			Message: "Instance is starting",
		}, nil
	case "running":
		return &provider.InfraStatus{
			State:   provider.StateRunning,
			Message: fmt.Sprintf("Instance running at %s", p.serverPublicIP),
		}, nil
	case "stopping", "stopped":
		return &provider.InfraStatus{
			State:   provider.StateStopped,
			Message: "Instance is stopped",
		}, nil
	case "shutting-down", "terminated":
		return &provider.InfraStatus{
			State:   provider.StateNotFound,
			Message: "Instance has been terminated",
		}, nil
	default:
		return &provider.InfraStatus{
			State:   provider.StateError,
			Message: fmt.Sprintf("Unknown instance state: %s", state),
		}, nil
	}
}

// EstimateCost returns an estimate of the hourly cost for the given config.
func (p *Provider) EstimateCost(cfg provider.ProvisionConfig) provider.CostEstimate {
	// Approximate hourly costs for common instance types (USD, us-east-1)
	// These are estimates and may vary by region
	costs := map[string]float64{
		"t3.micro":  0.0104,
		"t3.small":  0.0208,
		"t3.medium": 0.0416,
		"t2.micro":  0.0116,
		"t2.small":  0.023,
		"t2.medium": 0.0464,
	}

	rate, ok := costs[cfg.InstanceType]
	if !ok {
		// Default to t3.micro cost if instance type is unknown
		rate = 0.0104
	}

	return provider.CostEstimate{
		HourlyRate: rate,
		Currency:   "USD",
	}
}

// regionDisplayName returns a human-readable name for an AWS region.
func regionDisplayName(regionID string) string {
	names := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"af-south-1":     "Africa (Cape Town)",
		"ap-east-1":      "Asia Pacific (Hong Kong)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"ap-south-2":     "Asia Pacific (Hyderabad)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-southeast-3": "Asia Pacific (Jakarta)",
		"ap-southeast-4": "Asia Pacific (Melbourne)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-northeast-3": "Asia Pacific (Osaka)",
		"ca-central-1":   "Canada (Central)",
		"ca-west-1":      "Canada (Calgary)",
		"eu-central-1":   "Europe (Frankfurt)",
		"eu-central-2":   "Europe (Zurich)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-west-2":      "Europe (London)",
		"eu-west-3":      "Europe (Paris)",
		"eu-south-1":     "Europe (Milan)",
		"eu-south-2":     "Europe (Spain)",
		"eu-north-1":     "Europe (Stockholm)",
		"il-central-1":   "Israel (Tel Aviv)",
		"me-south-1":     "Middle East (Bahrain)",
		"me-central-1":   "Middle East (UAE)",
		"sa-east-1":      "South America (São Paulo)",
	}

	if name, ok := names[regionID]; ok {
		return name
	}
	return regionID
}

// Ensure Provider implements provider.Provider at compile time
var _ provider.Provider = (*Provider)(nil)
