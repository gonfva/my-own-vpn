// Package aws implements the cloud provider interface for Amazon Web Services.
package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

	// Track created resources for cleanup
	vpcID           string
	subnetID        string
	igwID           string
	routeTableID    string
	securityGroupID string
	keyPairName     string
	instanceID      string
}

// New creates a new AWS provider with the given credentials and region.
// If region is empty, it defaults to "us-east-1".
func New(ctx context.Context, accessKey, secretKey, region string) (*Provider, error) {
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"", // session token (empty for long-term credentials)
		)),
		// Disable shared config to avoid picking up local credentials
		config.WithSharedConfigProfile(""),
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
// This is a stub implementation that will be completed in subsequent issues.
func (p *Provider) Provision(ctx context.Context, config provider.ProvisionConfig) (*provider.ServerInfo, error) {
	// TODO: Implement in subsequent issues:
	// 1. Create VPC
	// 2. Create subnet
	// 3. Create Internet Gateway
	// 4. Create route table
	// 5. Create security group
	// 6. Create key pair
	// 7. Launch EC2 instance
	// 8. Wait for instance to be running
	// 9. Configure WireGuard on instance
	return nil, ErrNotImplemented
}

// Deprovision tears down all AWS infrastructure created by Provision.
// This is a stub implementation that will be completed in subsequent issues.
func (p *Provider) Deprovision(ctx context.Context) error {
	// TODO: Implement in subsequent issues:
	// 1. Terminate EC2 instance
	// 2. Delete key pair
	// 3. Delete security group
	// 4. Delete route table
	// 5. Detach and delete Internet Gateway
	// 6. Delete subnet
	// 7. Delete VPC
	if p.instanceID == "" {
		return ErrNotProvisioned
	}
	return ErrNotImplemented
}

// GetStatus returns the current status of the provisioned infrastructure.
// This is a stub implementation that will be completed in subsequent issues.
func (p *Provider) GetStatus(ctx context.Context) (*provider.InfraStatus, error) {
	if p.instanceID == "" {
		return &provider.InfraStatus{
			State:   provider.StateNotFound,
			Message: "No infrastructure provisioned",
		}, nil
	}

	// TODO: Implement in subsequent issues:
	// 1. Describe EC2 instance
	// 2. Map instance state to InfraStatus
	return nil, ErrNotImplemented
}

// EstimateCost returns an estimate of the hourly cost for the given config.
func (p *Provider) EstimateCost(config provider.ProvisionConfig) provider.CostEstimate {
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

	rate, ok := costs[config.InstanceType]
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
