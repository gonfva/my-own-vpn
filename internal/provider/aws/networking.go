package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Networking constants
const (
	// vpcCIDR is the CIDR block for the VPC
	vpcCIDR = "10.0.0.0/16"

	// subnetCIDR is the CIDR block for the public subnet
	subnetCIDR = "10.0.1.0/24"

	// wireGuardPort is the UDP port for WireGuard
	wireGuardPort = 51820

	// resourceTagName is the tag name used to identify resources created by this app
	resourceTagName = "my-own-vpn"

	// waitTimeout is the maximum time to wait for resource operations
	waitTimeout = 5 * time.Minute
)

// getTags returns the standard tags applied to all resources.
func (p *Provider) getTags() []types.Tag {
	tags := []types.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(resourceTagName),
		},
		{
			Key:   aws.String(tagKeyApplication),
			Value: aws.String(tagValueApplication),
		},
		{
			Key:   aws.String(tagKeyManagedBy),
			Value: aws.String(tagValueApplication),
		},
	}

	// Add session ID tag if available
	if p.sessionID != "" {
		tags = append(tags, types.Tag{
			Key:   aws.String(tagKeySessionID),
			Value: aws.String(p.sessionID),
		})
	}

	return tags
}

// getTagSpecifications returns TagSpecifications for a given resource type.
func (p *Provider) getTagSpecifications(resourceType types.ResourceType) []types.TagSpecification {
	return []types.TagSpecification{
		{
			ResourceType: resourceType,
			Tags:         p.getTags(),
		},
	}
}

// createVPC creates a VPC with the configured CIDR and enables DNS hostnames.
func (p *Provider) createVPC(ctx context.Context) error {
	result, err := p.ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock:         aws.String(vpcCIDR),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeVpc),
	})
	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}

	p.vpcID = *result.Vpc.VpcId

	// Wait for VPC to be available
	waiter := ec2.NewVpcAvailableWaiter(p.ec2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{p.vpcID},
	}, waitTimeout); err != nil {
		return fmt.Errorf("failed waiting for VPC to be available: %w", err)
	}

	// Enable DNS hostnames on the VPC
	_, err = p.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(p.vpcID),
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("failed to enable DNS hostnames on VPC: %w", err)
	}

	return nil
}

// createInternetGateway creates an Internet Gateway and attaches it to the VPC.
func (p *Provider) createInternetGateway(ctx context.Context) error {
	if p.vpcID == "" {
		return fmt.Errorf("VPC must be created before Internet Gateway")
	}

	result, err := p.ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeInternetGateway),
	})
	if err != nil {
		return fmt.Errorf("failed to create Internet Gateway: %w", err)
	}

	p.internetGatewayID = *result.InternetGateway.InternetGatewayId

	// Attach the Internet Gateway to the VPC
	_, err = p.ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(p.internetGatewayID),
		VpcId:             aws.String(p.vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to attach Internet Gateway to VPC: %w", err)
	}

	return nil
}

// createSubnet creates a public subnet in the VPC with auto-assign public IP enabled.
func (p *Provider) createSubnet(ctx context.Context) error {
	if p.vpcID == "" {
		return fmt.Errorf("VPC must be created before subnet")
	}

	result, err := p.ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:             aws.String(p.vpcID),
		CidrBlock:         aws.String(subnetCIDR),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeSubnet),
	})
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	p.subnetID = *result.Subnet.SubnetId

	// Wait for subnet to be available
	waiter := ec2.NewSubnetAvailableWaiter(p.ec2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{p.subnetID},
	}, waitTimeout); err != nil {
		return fmt.Errorf("failed waiting for subnet to be available: %w", err)
	}

	// Enable auto-assign public IP on the subnet
	_, err = p.ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		SubnetId:            aws.String(p.subnetID),
		MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("failed to enable auto-assign public IP on subnet: %w", err)
	}

	return nil
}

// createRouteTable creates a route table with a route to the Internet Gateway and associates it with the subnet.
func (p *Provider) createRouteTable(ctx context.Context) error {
	if p.vpcID == "" {
		return fmt.Errorf("VPC must be created before route table")
	}
	if p.internetGatewayID == "" {
		return fmt.Errorf("internet gateway must be created before route table")
	}
	if p.subnetID == "" {
		return fmt.Errorf("subnet must be created before route table")
	}

	// Create route table
	result, err := p.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId:             aws.String(p.vpcID),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeRouteTable),
	})
	if err != nil {
		return fmt.Errorf("failed to create route table: %w", err)
	}

	p.routeTableID = *result.RouteTable.RouteTableId

	// Add route for all traffic to Internet Gateway
	_, err = p.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(p.routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(p.internetGatewayID),
	})
	if err != nil {
		return fmt.Errorf("failed to create route to Internet Gateway: %w", err)
	}

	// Associate route table with subnet
	_, err = p.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(p.routeTableID),
		SubnetId:     aws.String(p.subnetID),
	})
	if err != nil {
		return fmt.Errorf("failed to associate route table with subnet: %w", err)
	}

	return nil
}

// createSecurityGroup creates a security group that allows WireGuard traffic.
func (p *Provider) createSecurityGroup(ctx context.Context) error {
	if p.vpcID == "" {
		return fmt.Errorf("VPC must be created before security group")
	}

	result, err := p.ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:         aws.String("my-own-vpn-wireguard"),
		Description:       aws.String("Security group for WireGuard VPN"),
		VpcId:             aws.String(p.vpcID),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeSecurityGroup),
	})
	if err != nil {
		return fmt.Errorf("failed to create security group: %w", err)
	}

	p.securityGroupID = *result.GroupId

	// Add inbound rule for WireGuard (UDP 51820)
	_, err = p.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(p.securityGroupID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("udp"),
				FromPort:   aws.Int32(wireGuardPort),
				ToPort:     aws.Int32(wireGuardPort),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("WireGuard VPN"),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add WireGuard ingress rule: %w", err)
	}

	// Note: Default security group allows all outbound traffic, which is what we need

	return nil
}

// createNetworking creates all networking infrastructure in the correct order.
func (p *Provider) createNetworking(ctx context.Context) error {
	if err := p.createVPC(ctx); err != nil {
		return err
	}

	if err := p.createInternetGateway(ctx); err != nil {
		return err
	}

	if err := p.createSubnet(ctx); err != nil {
		return err
	}

	if err := p.createRouteTable(ctx); err != nil {
		return err
	}

	if err := p.createSecurityGroup(ctx); err != nil {
		return err
	}

	return nil
}

// deleteSecurityGroup deletes the security group if it exists.
func (p *Provider) deleteSecurityGroup(ctx context.Context) error {
	if p.securityGroupID == "" {
		return nil
	}

	_, err := p.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(p.securityGroupID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}

	p.securityGroupID = ""
	return nil
}

// deleteRouteTable deletes the route table if it exists.
func (p *Provider) deleteRouteTable(ctx context.Context) error {
	if p.routeTableID == "" {
		return nil
	}

	// First, get route table associations to disassociate them
	describeResult, err := p.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{p.routeTableID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe route table: %w", err)
	}

	// Disassociate all associations (except the main one)
	for i := range describeResult.RouteTables {
		for j := range describeResult.RouteTables[i].Associations {
			assoc := &describeResult.RouteTables[i].Associations[j]
			if assoc.Main != nil && !*assoc.Main && assoc.RouteTableAssociationId != nil {
				_, err := p.ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				})
				if err != nil {
					return fmt.Errorf("failed to disassociate route table: %w", err)
				}
			}
		}
	}

	_, err = p.ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: aws.String(p.routeTableID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete route table: %w", err)
	}

	p.routeTableID = ""
	return nil
}

// deleteSubnet deletes the subnet if it exists.
func (p *Provider) deleteSubnet(ctx context.Context) error {
	if p.subnetID == "" {
		return nil
	}

	_, err := p.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: aws.String(p.subnetID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete subnet: %w", err)
	}

	p.subnetID = ""
	return nil
}

// deleteInternetGateway detaches and deletes the Internet Gateway if it exists.
func (p *Provider) deleteInternetGateway(ctx context.Context) error {
	if p.internetGatewayID == "" {
		return nil
	}

	// Detach from VPC first
	if p.vpcID != "" {
		_, err := p.ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: aws.String(p.internetGatewayID),
			VpcId:             aws.String(p.vpcID),
		})
		if err != nil {
			return fmt.Errorf("failed to detach Internet Gateway: %w", err)
		}
	}

	_, err := p.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(p.internetGatewayID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete Internet Gateway: %w", err)
	}

	p.internetGatewayID = ""
	return nil
}

// deleteVPC deletes the VPC if it exists.
func (p *Provider) deleteVPC(ctx context.Context) error {
	if p.vpcID == "" {
		return nil
	}

	_, err := p.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(p.vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VPC: %w", err)
	}

	p.vpcID = ""
	return nil
}

// deleteNetworking tears down all networking infrastructure in the correct order.
// It attempts to delete all resources even if some fail, collecting all errors.
func (p *Provider) deleteNetworking(ctx context.Context) error {
	var errs []error

	// Delete in reverse order of creation
	if err := p.deleteSecurityGroup(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := p.deleteRouteTable(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := p.deleteSubnet(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := p.deleteInternetGateway(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := p.deleteVPC(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete networking resources: %v", errs)
	}

	return nil
}

// NetworkingInfo contains information about created networking resources.
type NetworkingInfo struct {
	VPCID             string
	InternetGatewayID string
	SubnetID          string
	RouteTableID      string
	SecurityGroupID   string
}

// GetNetworkingInfo returns the IDs of all created networking resources.
func (p *Provider) GetNetworkingInfo() NetworkingInfo {
	return NetworkingInfo{
		VPCID:             p.vpcID,
		InternetGatewayID: p.internetGatewayID,
		SubnetID:          p.subnetID,
		RouteTableID:      p.routeTableID,
		SecurityGroupID:   p.securityGroupID,
	}
}
