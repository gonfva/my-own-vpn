package aws

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple session IDs and verify they are unique and well-formed
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID() returned error: %v", err)
		}

		// Session ID should be 16 characters (8 bytes hex encoded)
		if len(id) != 16 {
			t.Errorf("expected session ID length 16, got %d: %s", len(id), id)
		}

		// Verify uniqueness
		if ids[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGetStateFilePath(t *testing.T) {
	path, err := getStateFilePath()
	if err != nil {
		t.Fatalf("getStateFilePath() returned error: %v", err)
	}

	// Path should not be empty
	if path == "" {
		t.Error("expected non-empty path")
	}

	// Path should end with the state file name
	if filepath.Base(path) != stateFileName {
		t.Errorf("expected path to end with %s, got %s", stateFileName, filepath.Base(path))
	}

	// Directory should exist or be creatable
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Errorf("state directory does not exist: %v", err)
	} else if !info.IsDir() {
		t.Error("state path parent is not a directory")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	ctx := context.Background()

	// Create a provider with test data
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set some test state
	p.sessionID = "test-session-123"
	p.vpcID = "vpc-12345"
	p.internetGatewayID = "igw-12345"
	p.subnetID = "subnet-12345"
	p.routeTableID = "rtb-12345"
	p.securityGroupID = "sg-12345"
	p.instanceID = "i-12345"
	p.keyPairName = "test-keypair"
	p.serverPublicIP = "1.2.3.4"
	p.serverPublicKey = "test-public-key"

	// Save state
	if err := p.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}

	// Clean up state file after test
	defer func() { _ = clearState() }()

	// Load state into a new provider
	loadedState, err := loadState()
	if err != nil {
		t.Fatalf("loadState() returned error: %v", err)
	}
	if loadedState == nil {
		t.Fatal("loadState() returned nil")
	}

	// Verify all fields
	if loadedState.SessionID != "test-session-123" {
		t.Errorf("expected session ID test-session-123, got %s", loadedState.SessionID)
	}
	if loadedState.Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %s", loadedState.Region)
	}
	if loadedState.VPCID != "vpc-12345" {
		t.Errorf("expected VPC ID vpc-12345, got %s", loadedState.VPCID)
	}
	if loadedState.InternetGatewayID != "igw-12345" {
		t.Errorf("expected IGW ID igw-12345, got %s", loadedState.InternetGatewayID)
	}
	if loadedState.SubnetID != "subnet-12345" {
		t.Errorf("expected subnet ID subnet-12345, got %s", loadedState.SubnetID)
	}
	if loadedState.RouteTableID != "rtb-12345" {
		t.Errorf("expected route table ID rtb-12345, got %s", loadedState.RouteTableID)
	}
	if loadedState.SecurityGroupID != "sg-12345" {
		t.Errorf("expected security group ID sg-12345, got %s", loadedState.SecurityGroupID)
	}
	if loadedState.InstanceID != "i-12345" {
		t.Errorf("expected instance ID i-12345, got %s", loadedState.InstanceID)
	}
	if loadedState.KeyPairName != "test-keypair" {
		t.Errorf("expected key pair name test-keypair, got %s", loadedState.KeyPairName)
	}
	if loadedState.ServerPublicIP != "1.2.3.4" {
		t.Errorf("expected server IP 1.2.3.4, got %s", loadedState.ServerPublicIP)
	}
	if loadedState.ServerPublicKey != "test-public-key" {
		t.Errorf("expected server key test-public-key, got %s", loadedState.ServerPublicKey)
	}
}

func TestLoadStateNonExistent(t *testing.T) {
	// Ensure state file doesn't exist
	_ = clearState()

	state, err := loadState()
	if err != nil {
		t.Fatalf("loadState() returned error for non-existent file: %v", err)
	}
	if state != nil {
		t.Error("expected nil state for non-existent file")
	}
}

func TestClearState(t *testing.T) {
	ctx := context.Background()

	// Create a provider and save some state
	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	p.sessionID = "test-session"
	p.vpcID = "vpc-test"
	if err := p.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}

	// Clear state
	if err := clearState(); err != nil {
		t.Fatalf("clearState() returned error: %v", err)
	}

	// Verify state file is gone
	state, err := loadState()
	if err != nil {
		t.Fatalf("loadState() returned error: %v", err)
	}
	if state != nil {
		t.Error("expected nil state after clear")
	}
}

func TestClearStateNonExistent(t *testing.T) {
	// Ensure state file doesn't exist
	_ = clearState()

	// Clearing non-existent state should not error
	if err := clearState(); err != nil {
		t.Errorf("clearState() returned error for non-existent file: %v", err)
	}
}

func TestRestoreFromState(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	state := &ProvisionState{
		SessionID:         "restored-session",
		Region:            "us-east-1",
		VPCID:             "vpc-restored",
		InternetGatewayID: "igw-restored",
		SubnetID:          "subnet-restored",
		RouteTableID:      "rtb-restored",
		SecurityGroupID:   "sg-restored",
		InstanceID:        "i-restored",
		KeyPairName:       "restored-keypair",
		ServerPublicIP:    "5.6.7.8",
		ServerPublicKey:   "restored-key",
	}

	p.restoreFromState(state)

	if p.sessionID != "restored-session" {
		t.Errorf("expected session ID restored-session, got %s", p.sessionID)
	}
	if p.vpcID != "vpc-restored" {
		t.Errorf("expected VPC ID vpc-restored, got %s", p.vpcID)
	}
	if p.instanceID != "i-restored" {
		t.Errorf("expected instance ID i-restored, got %s", p.instanceID)
	}
	if p.serverPublicIP != "5.6.7.8" {
		t.Errorf("expected server IP 5.6.7.8, got %s", p.serverPublicIP)
	}
}

func TestHasProvisionedResources(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		instanceID string
		vpcID      string
		expected   bool
	}{
		{
			name:       "no resources",
			instanceID: "",
			vpcID:      "",
			expected:   false,
		},
		{
			name:       "only instance",
			instanceID: "i-12345",
			vpcID:      "",
			expected:   true,
		},
		{
			name:       "only vpc",
			instanceID: "",
			vpcID:      "vpc-12345",
			expected:   true,
		},
		{
			name:       "both resources",
			instanceID: "i-12345",
			vpcID:      "vpc-12345",
			expected:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
			if err != nil {
				t.Fatalf("New() returned error: %v", err)
			}

			p.instanceID = tc.instanceID
			p.vpcID = tc.vpcID

			result := p.HasProvisionedResources()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestLoadExistingStateRegionMismatch(t *testing.T) {
	ctx := context.Background()

	// Create a provider and save state for one region
	p1, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	p1.sessionID = "test-session"
	p1.vpcID = "vpc-test"
	if err := p1.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}
	defer func() { _ = clearState() }()

	// Create a provider for a different region
	p2, err := New(ctx, "test-access-key", "test-secret-key", "eu-west-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Loading state should return false due to region mismatch
	loaded, err := p2.LoadExistingState()
	if err != nil {
		t.Fatalf("LoadExistingState() returned error: %v", err)
	}
	if loaded {
		t.Error("expected loaded=false for region mismatch")
	}
	if p2.vpcID != "" {
		t.Errorf("expected empty VPC ID, got %s", p2.vpcID)
	}
}

func TestLoadExistingStateSuccess(t *testing.T) {
	ctx := context.Background()

	// Create a provider and save state
	p1, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	p1.sessionID = "test-session"
	p1.vpcID = "vpc-test"
	p1.instanceID = "i-test"
	if err := p1.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}
	defer func() { _ = clearState() }()

	// Create a new provider for the same region
	p2, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Loading state should succeed
	loaded, err := p2.LoadExistingState()
	if err != nil {
		t.Fatalf("LoadExistingState() returned error: %v", err)
	}
	if !loaded {
		t.Error("expected loaded=true")
	}
	if p2.vpcID != "vpc-test" {
		t.Errorf("expected VPC ID vpc-test, got %s", p2.vpcID)
	}
	if p2.instanceID != "i-test" {
		t.Errorf("expected instance ID i-test, got %s", p2.instanceID)
	}
}

func TestTagConstants(t *testing.T) {
	// Verify tag constants have expected values
	if tagKeyApplication != "Application" {
		t.Errorf("expected tagKeyApplication=Application, got %s", tagKeyApplication)
	}
	if tagKeyManagedBy != "ManagedBy" {
		t.Errorf("expected tagKeyManagedBy=ManagedBy, got %s", tagKeyManagedBy)
	}
	if tagKeySessionID != "SessionID" {
		t.Errorf("expected tagKeySessionID=SessionID, got %s", tagKeySessionID)
	}
	if tagValueApplication != "my-own-vpn" {
		t.Errorf("expected tagValueApplication=my-own-vpn, got %s", tagValueApplication)
	}
}
