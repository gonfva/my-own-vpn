package aws

import (
	"context"
	"os"
	"testing"

	"github.com/gonfva/my-own-vpn/internal/provider"
)

func TestGenerateUserData(t *testing.T) {
	userData := generateUserData()

	// Check that the user data contains expected WireGuard setup commands
	expectedContent := []string{
		"#!/bin/bash",
		"apt-get install -y wireguard",
		"wg genkey",
		"wg pubkey",
		"/etc/wireguard/wg0.conf",
		"ListenPort = 51820",
		"net.ipv4.ip_forward=1",
		"systemctl enable wg-quick@wg0",
		"systemctl start wg-quick@wg0",
		"WIREGUARD_READY=true",
		"WIREGUARD_PUBKEY=",
	}

	for _, expected := range expectedContent {
		if !contains(userData, expected) {
			t.Errorf("user data does not contain expected content: %s", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExtractWireGuardPubKey(t *testing.T) {
	// WireGuard public keys are 44 characters (base64 encoded 32 bytes)
	validKey := "abcdefghijklmnopqrstuvwxyz1234567890ABCDEF12"       // exactly 44 chars
	keyWithExtra := "abcdefghijklmnopqrstuvwxyz1234567890ABCDEF12xx" // 46 chars

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "valid public key",
			output:   "some output\nWIREGUARD_PUBKEY=" + validKey + "\nmore output",
			expected: validKey,
		},
		{
			name:     "valid public key with extra characters",
			output:   "WIREGUARD_PUBKEY=" + keyWithExtra,
			expected: validKey, // Should extract first 44 chars
		},
		{
			name:     "no public key",
			output:   "some output without the key",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "key too short",
			output:   "WIREGUARD_PUBKEY=short",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractWireGuardPubKey(tc.output)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestInstanceConstants(t *testing.T) {
	// Verify constants have reasonable values
	if defaultInstanceType != "t3.micro" {
		t.Errorf("expected default instance type t3.micro, got %s", defaultInstanceType)
	}

	if instanceWaitTimeout <= 0 {
		t.Error("instanceWaitTimeout should be positive")
	}

	if wireGuardReadyTimeout <= 0 {
		t.Error("wireGuardReadyTimeout should be positive")
	}

	if wireGuardPollInterval <= 0 {
		t.Error("wireGuardPollInterval should be positive")
	}
}

func TestGetInstanceInfoEmpty(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	info := p.GetInstanceInfo()
	if info.InstanceID != "" {
		t.Errorf("expected empty instance ID, got %s", info.InstanceID)
	}
	if info.KeyPairName != "" {
		t.Errorf("expected empty key pair name, got %s", info.KeyPairName)
	}
}

func TestLaunchInstanceRequiresSubnet(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.launchInstance(ctx, "t3.micro")
	if err == nil {
		t.Error("expected error when subnet is not created")
	}
}

func TestLaunchInstanceRequiresSecurityGroup(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set subnet ID but not security group
	p.subnetID = "subnet-12345"

	err = p.launchInstance(ctx, "t3.micro")
	if err == nil {
		t.Error("expected error when security group is not created")
	}
}

func TestTerminateInstanceEmptyIsNoop(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Should not error when instance ID is empty
	err = p.terminateInstance(ctx)
	if err != nil {
		t.Errorf("expected no error when terminating with empty instance ID, got: %v", err)
	}
}

func TestDeleteKeyPairEmptyIsNoop(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Should not error when key pair name is empty
	err = p.deleteKeyPair(ctx)
	if err != nil {
		t.Errorf("expected no error when deleting with empty key pair name, got: %v", err)
	}
}

func TestWaitForInstanceRunningNoInstance(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.waitForInstanceRunning(ctx)
	if err == nil {
		t.Error("expected error when no instance exists")
	}
}

func TestGetInstancePublicIPNoInstance(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	_, err = p.getInstancePublicIP(ctx)
	if err == nil {
		t.Error("expected error when no instance exists")
	}
}

func TestWaitForWireGuardReadyNoInstance(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	_, err = p.waitForWireGuardReady(ctx)
	if err == nil {
		t.Error("expected error when no instance exists")
	}
}

func TestGetInstanceStateNoInstance(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	_, err = p.getInstanceState(ctx)
	if err == nil {
		t.Error("expected error when no instance exists")
	}
}

// Integration test for full provision/deprovision cycle
// This test requires valid AWS credentials and will create actual resources
func TestProvisionAndDeprovisionIntegration(t *testing.T) {
	accessKey := os.Getenv("TEST_AWS_ACCESS_KEY")
	secretKey := os.Getenv("TEST_AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("TEST_AWS_ACCESS_KEY and TEST_AWS_SECRET_KEY not set")
	}

	// Skip in short mode as this test takes several minutes
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	p, err := New(ctx, accessKey, secretKey, "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Ensure cleanup even if test fails
	defer func() {
		if p.instanceID != "" || p.vpcID != "" {
			t.Log("Cleaning up resources...")
			if cleanupErr := p.Deprovision(ctx); cleanupErr != nil {
				t.Logf("Warning: cleanup failed: %v", cleanupErr)
			}
		}
	}()

	// Test provisioning
	t.Log("Starting provision...")
	serverInfo, err := p.Provision(ctx, provider.ProvisionConfig{
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})
	if err != nil {
		t.Fatalf("Provision() returned error: %v", err)
	}

	// Verify server info
	if serverInfo.PublicIP == "" {
		t.Error("expected public IP, got empty string")
	}
	if serverInfo.WireGuardPort != wireGuardPort {
		t.Errorf("expected WireGuard port %d, got %d", wireGuardPort, serverInfo.WireGuardPort)
	}
	if serverInfo.ServerPublicKey == "" {
		t.Error("expected server public key, got empty string")
	}
	if len(serverInfo.ServerPublicKey) != 44 {
		t.Errorf("expected public key length 44, got %d", len(serverInfo.ServerPublicKey))
	}

	t.Logf("Server provisioned: IP=%s, Key=%s", serverInfo.PublicIP, serverInfo.ServerPublicKey)

	// Test GetStatus
	status, err := p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() returned error: %v", err)
	}
	if status.State != provider.StateRunning {
		t.Errorf("expected state %s, got %s", provider.StateRunning, status.State)
	}

	// Test deprovisioning
	t.Log("Starting deprovision...")
	err = p.Deprovision(ctx)
	if err != nil {
		t.Fatalf("Deprovision() returned error: %v", err)
	}

	// Verify resources are cleaned up
	if p.instanceID != "" {
		t.Error("instance ID should be empty after deprovision")
	}
	if p.vpcID != "" {
		t.Error("VPC ID should be empty after deprovision")
	}

	t.Log("Integration test completed successfully")
}

// Test for getting Ubuntu AMI
func TestGetLatestUbuntuAMI(t *testing.T) {
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

	amiID, err := p.getLatestUbuntuAMI(ctx)
	if err != nil {
		t.Fatalf("getLatestUbuntuAMI() returned error: %v", err)
	}

	if amiID == "" {
		t.Error("expected AMI ID, got empty string")
	}

	// AWS AMI IDs start with "ami-"
	if len(amiID) < 4 || amiID[:4] != "ami-" {
		t.Errorf("expected AMI ID to start with 'ami-', got %s", amiID)
	}

	t.Logf("Found Ubuntu AMI: %s", amiID)
}
