package aws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ProvisionState stores the IDs of all provisioned resources for crash recovery.
// This state is persisted to disk so resources can be cleaned up even after app restart.
type ProvisionState struct {
	SessionID         string `json:"session_id"`
	Region            string `json:"region"`
	VPCID             string `json:"vpc_id,omitempty"`
	InternetGatewayID string `json:"internet_gateway_id,omitempty"`
	SubnetID          string `json:"subnet_id,omitempty"`
	RouteTableID      string `json:"route_table_id,omitempty"`
	SecurityGroupID   string `json:"security_group_id,omitempty"`
	InstanceID        string `json:"instance_id,omitempty"`
	KeyPairName       string `json:"key_pair_name,omitempty"`
	ServerPublicIP    string `json:"server_public_ip,omitempty"`
	ServerPublicKey   string `json:"server_public_key,omitempty"`
}

const (
	// stateFileName is the name of the state file
	stateFileName = "aws_provision_state.json"

	// tagKeyApplication is the tag key for application identification
	tagKeyApplication = "Application"

	// tagKeyManagedBy is the tag key for management identification
	tagKeyManagedBy = "ManagedBy"

	// tagKeySessionID is the tag key for session identification
	tagKeySessionID = "SessionID"

	// tagValueApplication is the value for the Application tag
	tagValueApplication = "my-own-vpn"
)

// generateSessionID creates a unique session identifier.
func generateSessionID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// getStateFilePath returns the path to the state file.
// It uses the user's config directory to store the state.
// The path is cleaned with filepath.Clean to prevent path traversal.
func getStateFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	appDir := filepath.Clean(filepath.Join(configDir, "my-own-vpn"))
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create app directory: %w", err)
	}

	return filepath.Clean(filepath.Join(appDir, stateFileName)), nil
}

// saveState persists the current provision state to disk.
func (p *Provider) saveState() error {
	state := ProvisionState{
		SessionID:         p.sessionID,
		Region:            p.region,
		VPCID:             p.vpcID,
		InternetGatewayID: p.internetGatewayID,
		SubnetID:          p.subnetID,
		RouteTableID:      p.routeTableID,
		SecurityGroupID:   p.securityGroupID,
		InstanceID:        p.instanceID,
		KeyPairName:       p.keyPairName,
		ServerPublicIP:    p.serverPublicIP,
		ServerPublicKey:   p.serverPublicKey,
	}

	filePath, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadState loads the provision state from disk if it exists.
func loadState() (*ProvisionState, error) {
	filePath, err := getStateFilePath()
	if err != nil {
		return nil, err
	}

	// #nosec G304 - filePath is constructed from system config dir and a hardcoded constant
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file exists
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ProvisionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// clearState removes the state file from disk.
func clearState() error {
	filePath, err := getStateFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}

// restoreFromState restores the provider's internal state from a ProvisionState.
func (p *Provider) restoreFromState(state *ProvisionState) {
	p.sessionID = state.SessionID
	p.vpcID = state.VPCID
	p.internetGatewayID = state.InternetGatewayID
	p.subnetID = state.SubnetID
	p.routeTableID = state.RouteTableID
	p.securityGroupID = state.SecurityGroupID
	p.instanceID = state.InstanceID
	p.keyPairName = state.KeyPairName
	p.serverPublicIP = state.ServerPublicIP
	p.serverPublicKey = state.ServerPublicKey
}

// findResourcesByTag discovers existing resources created by this application
// using the ManagedBy tag. This allows cleanup of orphaned resources.
func (p *Provider) findResourcesByTag(ctx context.Context) error {
	tagFilter := types.Filter{
		Name:   aws.String("tag:" + tagKeyManagedBy),
		Values: []string{tagValueApplication},
	}

	// Find VPCs
	vpcs, err := p.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe VPCs: %w", err)
	}
	if len(vpcs.Vpcs) > 0 {
		p.vpcID = *vpcs.Vpcs[0].VpcId
		// Extract session ID from tags if available
		for _, tag := range vpcs.Vpcs[0].Tags {
			if tag.Key != nil && *tag.Key == tagKeySessionID && tag.Value != nil {
				p.sessionID = *tag.Value
			}
		}
	}

	// Find Internet Gateways
	igws, err := p.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe internet gateways: %w", err)
	}
	if len(igws.InternetGateways) > 0 {
		p.internetGatewayID = *igws.InternetGateways[0].InternetGatewayId
	}

	// Find Subnets
	subnets, err := p.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe subnets: %w", err)
	}
	if len(subnets.Subnets) > 0 {
		p.subnetID = *subnets.Subnets[0].SubnetId
	}

	// Find Route Tables (excluding main route tables)
	routeTables, err := p.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe route tables: %w", err)
	}
	for i := range routeTables.RouteTables {
		rt := &routeTables.RouteTables[i]
		// Skip main route tables
		isMain := false
		for j := range rt.Associations {
			assoc := &rt.Associations[j]
			if assoc.Main != nil && *assoc.Main {
				isMain = true
				break
			}
		}
		if !isMain && rt.RouteTableId != nil {
			p.routeTableID = *rt.RouteTableId
			break
		}
	}

	// Find Security Groups (excluding default)
	sgs, err := p.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe security groups: %w", err)
	}
	for i := range sgs.SecurityGroups {
		sg := &sgs.SecurityGroups[i]
		if sg.GroupName != nil && *sg.GroupName != "default" && sg.GroupId != nil {
			p.securityGroupID = *sg.GroupId
			break
		}
	}

	// Find EC2 Instances (running or stopped)
	instances, err := p.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			tagFilter,
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to describe instances: %w", err)
	}
	for i := range instances.Reservations {
		reservation := &instances.Reservations[i]
		for j := range reservation.Instances {
			instance := &reservation.Instances[j]
			if instance.InstanceId != nil {
				p.instanceID = *instance.InstanceId
				if instance.PublicIpAddress != nil {
					p.serverPublicIP = *instance.PublicIpAddress
				}
				break
			}
		}
	}

	// Find Key Pairs
	keyPairs, err := p.ec2Client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		Filters: []types.Filter{tagFilter},
	})
	if err != nil {
		return fmt.Errorf("failed to describe key pairs: %w", err)
	}
	if len(keyPairs.KeyPairs) > 0 && keyPairs.KeyPairs[0].KeyName != nil {
		p.keyPairName = *keyPairs.KeyPairs[0].KeyName
	}

	return nil
}

// HasProvisionedResources checks if there are any provisioned resources
// either in memory or persisted in the state file.
func (p *Provider) HasProvisionedResources() bool {
	return p.instanceID != "" || p.vpcID != ""
}

// LoadExistingState attempts to load existing state from disk and restore it.
// Returns true if state was found and loaded.
func (p *Provider) LoadExistingState() (bool, error) {
	state, err := loadState()
	if err != nil {
		return false, err
	}
	if state == nil {
		return false, nil
	}

	// Only restore if the state is for the same region
	if state.Region != p.region {
		return false, nil
	}

	p.restoreFromState(state)
	return true, nil
}
