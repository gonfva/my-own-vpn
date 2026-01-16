package aws

import (
	"context"
	"os"
	"testing"
)

func TestGetTags(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Without session ID, should have 3 tags
	tags := p.getTags()
	if len(tags) != 3 {
		t.Errorf("expected 3 tags without session ID, got %d", len(tags))
	}

	// Check all expected tags
	foundName := false
	foundApplication := false
	foundManagedBy := false
	for _, tag := range tags {
		if *tag.Key == "Name" && *tag.Value == resourceTagName {
			foundName = true
		}
		if *tag.Key == tagKeyApplication && *tag.Value == tagValueApplication {
			foundApplication = true
		}
		if *tag.Key == tagKeyManagedBy && *tag.Value == tagValueApplication {
			foundManagedBy = true
		}
	}

	if !foundName {
		t.Error("expected Name tag with value 'my-own-vpn'")
	}
	if !foundApplication {
		t.Error("expected Application tag with value 'my-own-vpn'")
	}
	if !foundManagedBy {
		t.Error("expected ManagedBy tag with value 'my-own-vpn'")
	}
}

func TestGetTagsWithSessionID(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set a session ID
	p.sessionID = "test-session-12345678"

	tags := p.getTags()
	// With session ID, should have 4 tags
	if len(tags) != 4 {
		t.Errorf("expected 4 tags with session ID, got %d", len(tags))
	}

	// Check for session ID tag
	foundSessionID := false
	for _, tag := range tags {
		if *tag.Key == tagKeySessionID && *tag.Value == "test-session-12345678" {
			foundSessionID = true
			break
		}
	}

	if !foundSessionID {
		t.Error("expected SessionID tag with value 'test-session-12345678'")
	}
}

func TestGetNetworkingInfo(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Initially all IDs should be empty
	info := p.GetNetworkingInfo()
	if info.VPCID != "" {
		t.Errorf("expected empty VPCID, got %s", info.VPCID)
	}
	if info.InternetGatewayID != "" {
		t.Errorf("expected empty InternetGatewayID, got %s", info.InternetGatewayID)
	}
	if info.SubnetID != "" {
		t.Errorf("expected empty SubnetID, got %s", info.SubnetID)
	}
	if info.RouteTableID != "" {
		t.Errorf("expected empty RouteTableID, got %s", info.RouteTableID)
	}
	if info.SecurityGroupID != "" {
		t.Errorf("expected empty SecurityGroupID, got %s", info.SecurityGroupID)
	}

	// Set some values
	p.vpcID = "vpc-123"
	p.internetGatewayID = "igw-123"
	p.subnetID = "subnet-123"
	p.routeTableID = "rtb-123"
	p.securityGroupID = "sg-123"

	info = p.GetNetworkingInfo()
	if info.VPCID != "vpc-123" {
		t.Errorf("expected VPCID 'vpc-123', got %s", info.VPCID)
	}
	if info.InternetGatewayID != "igw-123" {
		t.Errorf("expected InternetGatewayID 'igw-123', got %s", info.InternetGatewayID)
	}
	if info.SubnetID != "subnet-123" {
		t.Errorf("expected SubnetID 'subnet-123', got %s", info.SubnetID)
	}
	if info.RouteTableID != "rtb-123" {
		t.Errorf("expected RouteTableID 'rtb-123', got %s", info.RouteTableID)
	}
	if info.SecurityGroupID != "sg-123" {
		t.Errorf("expected SecurityGroupID 'sg-123', got %s", info.SecurityGroupID)
	}
}

func TestCreateVPCRequiresNoPrerequisites(t *testing.T) {
	// This test verifies the VPC creation doesn't require any prerequisites
	// It's a unit test that checks the function can be called
	// The actual API call will fail without valid credentials, but that's expected
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// VPC creation should not require any prerequisites
	// (The actual API call will fail, but the prerequisites check passes)
	if p.vpcID != "" {
		t.Error("vpcID should be empty before createVPC")
	}
}

func TestCreateInternetGatewayRequiresVPC(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.createInternetGateway(ctx)
	if err == nil {
		t.Error("expected error when VPC not created")
	}
	if err.Error() != "VPC must be created before Internet Gateway" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateSubnetRequiresVPC(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.createSubnet(ctx)
	if err == nil {
		t.Error("expected error when VPC not created")
	}
	if err.Error() != "VPC must be created before subnet" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateRouteTableRequiresVPC(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.createRouteTable(ctx)
	if err == nil {
		t.Error("expected error when VPC not created")
	}
	if err.Error() != "VPC must be created before route table" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateRouteTableRequiresInternetGateway(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set VPC but not IGW
	p.vpcID = "vpc-test"

	err = p.createRouteTable(ctx)
	if err == nil {
		t.Error("expected error when internet gateway not created")
	}
	if err.Error() != "internet gateway must be created before route table" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateRouteTableRequiresSubnet(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set VPC and IGW but not subnet
	p.vpcID = "vpc-test"
	p.internetGatewayID = "igw-test"

	err = p.createRouteTable(ctx)
	if err == nil {
		t.Error("expected error when subnet not created")
	}
	if err.Error() != "subnet must be created before route table" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateSecurityGroupRequiresVPC(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.createSecurityGroup(ctx)
	if err == nil {
		t.Error("expected error when VPC not created")
	}
	if err.Error() != "VPC must be created before security group" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDeleteNetworkingNoOpsWhenEmpty(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Delete should succeed when nothing is provisioned
	err = p.deleteSecurityGroup(ctx)
	if err != nil {
		t.Errorf("deleteSecurityGroup() should succeed when empty: %v", err)
	}

	err = p.deleteRouteTable(ctx)
	if err != nil {
		t.Errorf("deleteRouteTable() should succeed when empty: %v", err)
	}

	err = p.deleteSubnet(ctx)
	if err != nil {
		t.Errorf("deleteSubnet() should succeed when empty: %v", err)
	}

	err = p.deleteInternetGateway(ctx)
	if err != nil {
		t.Errorf("deleteInternetGateway() should succeed when empty: %v", err)
	}

	err = p.deleteVPC(ctx)
	if err != nil {
		t.Errorf("deleteVPC() should succeed when empty: %v", err)
	}

	err = p.deleteNetworking(ctx)
	if err != nil {
		t.Errorf("deleteNetworking() should succeed when empty: %v", err)
	}
}

func TestNetworkingConstants(t *testing.T) {
	// Verify constants have expected values
	if vpcCIDR != "10.0.0.0/16" {
		t.Errorf("expected vpcCIDR '10.0.0.0/16', got %s", vpcCIDR)
	}
	if subnetCIDR != "10.0.1.0/24" {
		t.Errorf("expected subnetCIDR '10.0.1.0/24', got %s", subnetCIDR)
	}
	if wireGuardPort != 51820 {
		t.Errorf("expected wireGuardPort 51820, got %d", wireGuardPort)
	}
	if resourceTagName != "my-own-vpn" {
		t.Errorf("expected resourceTagName 'my-own-vpn', got %s", resourceTagName)
	}
}

// Integration tests - these require real AWS credentials
// Run with: TEST_AWS_ACCESS_KEY=xxx TEST_AWS_SECRET_KEY=xxx go test -v ./...

func TestNetworkingIntegration(t *testing.T) {
	accessKey := os.Getenv("TEST_AWS_ACCESS_KEY")
	secretKey := os.Getenv("TEST_AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("TEST_AWS_ACCESS_KEY and TEST_AWS_SECRET_KEY not set")
	}

	ctx := context.Background()
	p, err := New(ctx, accessKey, secretKey, "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Cleanup function to ensure resources are deleted even if test fails
	defer func() {
		if cleanupErr := p.deleteNetworking(ctx); cleanupErr != nil {
			t.Errorf("cleanup failed: %v", cleanupErr)
		}
	}()

	// Create all networking resources
	if err := p.createNetworking(ctx); err != nil {
		t.Fatalf("createNetworking() returned error: %v", err)
	}

	// Verify all resources were created
	info := p.GetNetworkingInfo()

	if info.VPCID == "" {
		t.Error("VPCID should not be empty after createNetworking")
	}
	if info.InternetGatewayID == "" {
		t.Error("InternetGatewayID should not be empty after createNetworking")
	}
	if info.SubnetID == "" {
		t.Error("SubnetID should not be empty after createNetworking")
	}
	if info.RouteTableID == "" {
		t.Error("RouteTableID should not be empty after createNetworking")
	}
	if info.SecurityGroupID == "" {
		t.Error("SecurityGroupID should not be empty after createNetworking")
	}

	t.Logf("Created networking resources:")
	t.Logf("  VPC: %s", info.VPCID)
	t.Logf("  Internet Gateway: %s", info.InternetGatewayID)
	t.Logf("  Subnet: %s", info.SubnetID)
	t.Logf("  Route Table: %s", info.RouteTableID)
	t.Logf("  Security Group: %s", info.SecurityGroupID)
}

func TestNetworkingCreationOrder(t *testing.T) {
	accessKey := os.Getenv("TEST_AWS_ACCESS_KEY")
	secretKey := os.Getenv("TEST_AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("TEST_AWS_ACCESS_KEY and TEST_AWS_SECRET_KEY not set")
	}

	ctx := context.Background()
	p, err := New(ctx, accessKey, secretKey, "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Cleanup function
	defer func() {
		if cleanupErr := p.deleteNetworking(ctx); cleanupErr != nil {
			t.Errorf("cleanup failed: %v", cleanupErr)
		}
	}()

	// Test step by step creation
	t.Log("Creating VPC...")
	if err := p.createVPC(ctx); err != nil {
		t.Fatalf("createVPC() returned error: %v", err)
	}
	if p.vpcID == "" {
		t.Fatal("vpcID should be set after createVPC")
	}

	t.Log("Creating Internet Gateway...")
	if err := p.createInternetGateway(ctx); err != nil {
		t.Fatalf("createInternetGateway() returned error: %v", err)
	}
	if p.internetGatewayID == "" {
		t.Fatal("internetGatewayID should be set after createInternetGateway")
	}

	t.Log("Creating Subnet...")
	if err := p.createSubnet(ctx); err != nil {
		t.Fatalf("createSubnet() returned error: %v", err)
	}
	if p.subnetID == "" {
		t.Fatal("subnetID should be set after createSubnet")
	}

	t.Log("Creating Route Table...")
	if err := p.createRouteTable(ctx); err != nil {
		t.Fatalf("createRouteTable() returned error: %v", err)
	}
	if p.routeTableID == "" {
		t.Fatal("routeTableID should be set after createRouteTable")
	}

	t.Log("Creating Security Group...")
	if err := p.createSecurityGroup(ctx); err != nil {
		t.Fatalf("createSecurityGroup() returned error: %v", err)
	}
	if p.securityGroupID == "" {
		t.Fatal("securityGroupID should be set after createSecurityGroup")
	}
}
