package aws

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/gonfva/my-own-vpn/internal/provider"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil provider")
	}

	// Default region should be us-east-1
	if p.region != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %s", p.region)
	}
}

func TestNewProviderWithRegion(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "eu-west-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil provider")
	}
	if p.region != "eu-west-1" {
		t.Errorf("expected region eu-west-1, got %s", p.region)
	}
}

func TestValidateCredentialsInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	p, err := New(ctx, "INVALID_ACCESS_KEY", "invalid_secret_key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.ValidateCredentials(ctx)
	if err == nil {
		t.Error("expected error for invalid credentials")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidateCredentialsValid(t *testing.T) {
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

	err = p.ValidateCredentials(ctx)
	if err != nil {
		t.Errorf("ValidateCredentials() returned error: %v", err)
	}
}

func TestListRegions(t *testing.T) {
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

	regions, err := p.ListRegions(ctx)
	if err != nil {
		t.Fatalf("ListRegions() returned error: %v", err)
	}

	if len(regions) == 0 {
		t.Error("ListRegions() returned no regions")
	}

	// Check that us-east-1 is in the list
	found := false
	for _, r := range regions {
		if r.ID == "us-east-1" {
			found = true
			if r.Name != "US East (N. Virginia)" {
				t.Errorf("expected name 'US East (N. Virginia)', got %s", r.Name)
			}
			break
		}
	}
	if !found {
		t.Error("us-east-1 not found in regions list")
	}
}

func TestProvisionFailsWithInvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	p, err := New(ctx, "INVALID_ACCESS_KEY", "invalid_secret_key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Provision should fail due to invalid credentials when trying to create VPC
	_, err = p.Provision(ctx, provider.ProvisionConfig{
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	})
	if err == nil {
		t.Error("expected error for invalid credentials during provision")
	}
}

func TestDeprovisionNotProvisioned(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	err = p.Deprovision(ctx)
	if !errors.Is(err, ErrNotProvisioned) {
		t.Errorf("expected ErrNotProvisioned, got: %v", err)
	}
}

func TestGetStatusNotProvisioned(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	status, err := p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() returned error: %v", err)
	}
	if status.State != provider.StateNotFound {
		t.Errorf("expected state %s, got %s", provider.StateNotFound, status.State)
	}
}

func TestEstimateCost(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tests := []struct {
		instanceType string
		expectedRate float64
	}{
		{"t3.micro", 0.0104},
		{"t3.small", 0.0208},
		{"t3.medium", 0.0416},
		{"t2.micro", 0.0116},
		{"unknown-type", 0.0104}, // defaults to t3.micro rate
	}

	for _, tc := range tests {
		t.Run(tc.instanceType, func(t *testing.T) {
			cost := p.EstimateCost(provider.ProvisionConfig{
				Region:       "us-east-1",
				InstanceType: tc.instanceType,
			})

			if cost.HourlyRate != tc.expectedRate {
				t.Errorf("expected rate %f, got %f", tc.expectedRate, cost.HourlyRate)
			}
			if cost.Currency != "USD" {
				t.Errorf("expected currency USD, got %s", cost.Currency)
			}
		})
	}
}

func TestRegionDisplayName(t *testing.T) {
	tests := []struct {
		regionID string
		expected string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"eu-west-1", "Europe (Ireland)"},
		{"ap-northeast-1", "Asia Pacific (Tokyo)"},
		{"unknown-region", "unknown-region"}, // unknown regions return ID
	}

	for _, tc := range tests {
		t.Run(tc.regionID, func(t *testing.T) {
			name := regionDisplayName(tc.regionID)
			if name != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, name)
			}
		})
	}
}

func TestProviderImplementsInterface(t *testing.T) {
	// This is a compile-time check, but we include it as a test for clarity
	var _ provider.Provider = (*Provider)(nil)
}

func TestGetSessionID(t *testing.T) {
	ctx := context.Background()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Initially session ID should be empty
	if p.GetSessionID() != "" {
		t.Error("expected empty session ID initially")
	}

	// Set a session ID
	p.sessionID = "test-session-12345678"

	// Verify getter returns it
	if p.GetSessionID() != "test-session-12345678" {
		t.Errorf("expected session ID test-session-12345678, got %s", p.GetSessionID())
	}
}

func TestCleanupOrphanedNoResources(t *testing.T) {
	ctx := context.Background()

	// Ensure no state file exists
	_ = clearState()

	p, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// CleanupOrphaned should return nil when there are no resources
	// Note: This will try to call findResourcesByTag which will fail with invalid creds
	// but since we're in short mode, the test should be skipped for actual AWS calls
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	err = p.CleanupOrphaned(ctx)
	// Error is expected because we have invalid credentials
	if err == nil {
		t.Log("CleanupOrphaned completed without error (no resources found)")
	}
}

func TestCleanupOrphanedWithPersistedState(t *testing.T) {
	ctx := context.Background()

	// Create a provider and save some state
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

	// Create a new provider
	p2, err := New(ctx, "test-access-key", "test-secret-key", "us-east-1")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Verify the provider has no resources loaded initially
	if p2.HasProvisionedResources() {
		t.Error("expected no provisioned resources initially")
	}

	// Skip actual cleanup as it requires valid credentials
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// CleanupOrphaned would load state and try to clean up
	// With invalid credentials, this will fail at the AWS API calls
	err = p2.CleanupOrphaned(ctx)
	// We expect an error because we're using invalid credentials
	if err != nil {
		t.Logf("CleanupOrphaned failed as expected with invalid credentials: %v", err)
	}
}
