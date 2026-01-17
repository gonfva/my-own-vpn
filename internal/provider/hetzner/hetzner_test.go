package hetzner

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/gonfva/my-own-vpn/internal/provider"
)

func TestNewProvider(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil provider")
	}
	if p.client == nil {
		t.Fatal("New() provider has nil client")
	}
}

func TestNewProviderEmptyToken(t *testing.T) {
	p, err := New("")
	if err == nil {
		t.Error("expected error for empty token")
	}
	if p != nil {
		t.Error("expected nil provider for empty token")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidateCredentialsInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p, err := New("invalid-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = p.ValidateCredentials(ctx)
	if err == nil {
		t.Error("expected error for invalid credentials")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidateCredentialsValid(t *testing.T) {
	token := os.Getenv("TEST_HETZNER_TOKEN")
	if token == "" {
		t.Skip("TEST_HETZNER_TOKEN not set")
	}

	p, err := New(token)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = p.ValidateCredentials(ctx)
	if err != nil {
		t.Errorf("ValidateCredentials() returned error: %v", err)
	}
}

func TestListRegions(t *testing.T) {
	token := os.Getenv("TEST_HETZNER_TOKEN")
	if token == "" {
		t.Skip("TEST_HETZNER_TOKEN not set")
	}

	p, err := New(token)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	regions, err := p.ListRegions(ctx)
	if err != nil {
		t.Fatalf("ListRegions() returned error: %v", err)
	}

	if len(regions) == 0 {
		t.Error("ListRegions() returned no regions")
	}

	// Check that fsn1 (Falkenstein) is in the list
	found := false
	for _, r := range regions {
		if r.ID == "fsn1" {
			found = true
			if r.Name != "Falkenstein, Germany" {
				t.Errorf("expected name 'Falkenstein, Germany', got %s", r.Name)
			}
			break
		}
	}
	if !found {
		t.Error("fsn1 not found in regions list")
	}
}

func TestProvisionFailsWithInvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p, err := New("invalid-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	// Provision should fail due to invalid credentials
	_, err = p.Provision(ctx, provider.ProvisionConfig{
		Region:       "fsn1",
		InstanceType: "cx11",
	})
	if err == nil {
		t.Error("expected error for invalid credentials during provision")
	}
}

func TestDeprovisionNotProvisioned(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = p.Deprovision(ctx)
	if !errors.Is(err, ErrNotProvisioned) {
		t.Errorf("expected ErrNotProvisioned, got: %v", err)
	}
}

func TestGetStatusNotProvisioned(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	status, err := p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() returned error: %v", err)
	}
	if status.State != provider.StateNotFound {
		t.Errorf("expected state %s, got %s", provider.StateNotFound, status.State)
	}
}

func TestEstimateCost(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tests := []struct {
		instanceType string
		expectedRate float64
	}{
		{"cx11", 0.0050},
		{"cx21", 0.0088},
		{"cx31", 0.0159},
		{"cpx11", 0.0062},
		{"cax11", 0.0053},
		{"unknown-type", 0.0050}, // defaults to cx11 rate
	}

	for _, tc := range tests {
		t.Run(tc.instanceType, func(t *testing.T) {
			cost := p.EstimateCost(provider.ProvisionConfig{
				Region:       "fsn1",
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

func TestLocationDisplayName(t *testing.T) {
	tests := []struct {
		locationID string
		city       string
		expected   string
	}{
		{"fsn1", "Falkenstein", "Falkenstein, Germany"},
		{"nbg1", "Nuremberg", "Nuremberg, Germany"},
		{"hel1", "Helsinki", "Helsinki, Finland"},
		{"ash", "Ashburn", "Ashburn, USA"},
		{"unknown-location", "Some City", "Some City"},
		{"unknown-location", "", "unknown-location"},
	}

	for _, tc := range tests {
		t.Run(tc.locationID, func(t *testing.T) {
			name := locationDisplayName(tc.locationID, tc.city)
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
	p, err := New("test-token")
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

func TestHasProvisionedResources(t *testing.T) {
	tests := []struct {
		name       string
		serverID   int64
		firewallID int64
		sshKeyID   int64
		expected   bool
	}{
		{"no resources", 0, 0, 0, false},
		{"server only", 123, 0, 0, true},
		{"firewall only", 0, 456, 0, true},
		{"ssh key only", 0, 0, 789, true},
		{"all resources", 123, 456, 789, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := New("test-token")
			if err != nil {
				t.Fatalf("New() returned error: %v", err)
			}
			p.serverID = tc.serverID
			p.firewallID = tc.firewallID
			p.sshKeyID = tc.sshKeyID

			if p.HasProvisionedResources() != tc.expected {
				t.Errorf("expected HasProvisionedResources() to be %v", tc.expected)
			}
		})
	}
}

func TestCleanupOrphanedNoResources(t *testing.T) {
	// Ensure no state file exists
	_ = clearState()

	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	err = p.CleanupOrphaned(ctx)
	// Error is expected because we have invalid credentials
	if err == nil {
		t.Log("CleanupOrphaned completed without error (no resources found)")
	}
}

func TestCleanupOrphanedWithPersistedState(t *testing.T) {
	// Create a provider and save some state
	p1, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	p1.sessionID = "test-session"
	p1.serverID = 12345
	if err := p1.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}
	defer func() { _ = clearState() }()

	// Create a new provider
	p2, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Verify the provider has no resources loaded initially
	if p2.HasProvisionedResources() {
		t.Error("expected no provisioned resources initially")
	}

	// Load existing state
	loaded, err := p2.LoadExistingState()
	if err != nil {
		t.Fatalf("LoadExistingState() returned error: %v", err)
	}
	if !loaded {
		t.Error("expected state to be loaded")
	}

	// Verify the state was restored
	if p2.sessionID != "test-session" {
		t.Errorf("expected session ID test-session, got %s", p2.sessionID)
	}
	if p2.serverID != 12345 {
		t.Errorf("expected server ID 12345, got %d", p2.serverID)
	}
}

func TestGenerateSessionID(t *testing.T) {
	sessionID1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() returned error: %v", err)
	}

	if len(sessionID1) != 16 {
		t.Errorf("expected session ID length 16, got %d", len(sessionID1))
	}

	// Generate another and ensure they're different
	sessionID2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() returned error: %v", err)
	}

	if sessionID1 == sessionID2 {
		t.Error("expected different session IDs")
	}
}

func TestGetLabels(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Without session ID
	labels := p.getLabels()
	if labels[labelKeyApplication] != labelValueApplication {
		t.Errorf("expected application label %s, got %s", labelValueApplication, labels[labelKeyApplication])
	}
	if labels[labelKeyManagedBy] != labelValueApplication {
		t.Errorf("expected managed-by label %s, got %s", labelValueApplication, labels[labelKeyManagedBy])
	}
	if _, ok := labels[labelKeySessionID]; ok {
		t.Error("expected no session-id label when session ID is empty")
	}

	// With session ID
	p.sessionID = "test-session-123"
	labels = p.getLabels()
	if labels[labelKeySessionID] != "test-session-123" {
		t.Errorf("expected session-id label test-session-123, got %s", labels[labelKeySessionID])
	}
}

func TestStatePersistence(t *testing.T) {
	// Clean up any existing state
	_ = clearState()
	defer func() { _ = clearState() }()

	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set some state
	p.sessionID = "test-session"
	p.serverID = 111
	p.firewallID = 222
	p.sshKeyID = 333
	p.serverPublicIP = "1.2.3.4"
	p.serverPublicKey = "test-pubkey-12345678901234567890123456789012"

	// Save state
	if err := p.saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}

	// Load state
	state, err := loadState()
	if err != nil {
		t.Fatalf("loadState() returned error: %v", err)
	}
	if state == nil {
		t.Fatal("loadState() returned nil state")
	}

	// Verify state
	if state.SessionID != "test-session" {
		t.Errorf("expected session ID test-session, got %s", state.SessionID)
	}
	if state.ServerID != 111 {
		t.Errorf("expected server ID 111, got %d", state.ServerID)
	}
	if state.FirewallID != 222 {
		t.Errorf("expected firewall ID 222, got %d", state.FirewallID)
	}
	if state.SSHKeyID != 333 {
		t.Errorf("expected SSH key ID 333, got %d", state.SSHKeyID)
	}
	if state.ServerPublicIP != "1.2.3.4" {
		t.Errorf("expected server public IP 1.2.3.4, got %s", state.ServerPublicIP)
	}
	if state.ServerPublicKey != "test-pubkey-12345678901234567890123456789012" {
		t.Errorf("expected server public key, got %s", state.ServerPublicKey)
	}

	// Clear state
	if err := clearState(); err != nil {
		t.Fatalf("clearState() returned error: %v", err)
	}

	// Verify state is cleared
	state, err = loadState()
	if err != nil {
		t.Fatalf("loadState() after clear returned error: %v", err)
	}
	if state != nil {
		t.Error("expected nil state after clear")
	}
}

func TestRestoreFromState(t *testing.T) {
	p, err := New("test-token")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	state := &ProvisionState{
		SessionID:       "test-session",
		ServerID:        111,
		FirewallID:      222,
		SSHKeyID:        333,
		ServerPublicIP:  "5.6.7.8",
		ServerPublicKey: "test-pubkey",
	}

	p.restoreFromState(state)

	if p.sessionID != "test-session" {
		t.Errorf("expected session ID test-session, got %s", p.sessionID)
	}
	if p.serverID != 111 {
		t.Errorf("expected server ID 111, got %d", p.serverID)
	}
	if p.firewallID != 222 {
		t.Errorf("expected firewall ID 222, got %d", p.firewallID)
	}
	if p.sshKeyID != 333 {
		t.Errorf("expected SSH key ID 333, got %d", p.sshKeyID)
	}
	if p.serverPublicIP != "5.6.7.8" {
		t.Errorf("expected server public IP 5.6.7.8, got %s", p.serverPublicIP)
	}
	if p.serverPublicKey != "test-pubkey" {
		t.Errorf("expected server public key test-pubkey, got %s", p.serverPublicKey)
	}
}
