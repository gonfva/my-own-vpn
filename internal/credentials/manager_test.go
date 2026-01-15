package credentials

import (
	"context"
	"errors"
	"testing"
)

// MockManager is a mock implementation of Manager for testing
type MockManager struct {
	awsCreds     *AWSCredentials
	hetznerCreds *HetznerCredentials
	saveError    error
	loadError    error
	deleteError  error
}

// NewMockManager creates a new MockManager instance
func NewMockManager() *MockManager {
	return &MockManager{}
}

// SetError sets the error to be returned by all operations
func (m *MockManager) SetError(err error) {
	m.saveError = err
	m.loadError = err
	m.deleteError = err
}

func (m *MockManager) SaveAWS(_ context.Context, creds AWSCredentials) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.awsCreds = &creds
	return nil
}

func (m *MockManager) LoadAWS(_ context.Context) (*AWSCredentials, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.awsCreds, nil
}

func (m *MockManager) DeleteAWS(_ context.Context) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	m.awsCreds = nil
	return nil
}

func (m *MockManager) SaveHetzner(_ context.Context, creds HetznerCredentials) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.hetznerCreds = &creds
	return nil
}

func (m *MockManager) LoadHetzner(_ context.Context) (*HetznerCredentials, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.hetznerCreds, nil
}

func (m *MockManager) DeleteHetzner(_ context.Context) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	m.hetznerCreds = nil
	return nil
}

func (m *MockManager) HasCredentials(_ context.Context, provider string) bool {
	switch provider {
	case ProviderAWS:
		return m.awsCreds != nil && !m.awsCreds.IsEmpty()
	case ProviderHetzner:
		return m.hetznerCreds != nil && !m.hetznerCreds.IsEmpty()
	default:
		return false
	}
}

// TestMockManagerImplementsInterface verifies MockManager implements Manager
func TestMockManagerImplementsInterface(t *testing.T) {
	var _ Manager = &MockManager{}
}

func TestManagerAWSOperations(t *testing.T) {
	ctx := context.Background()
	mgr := NewMockManager()

	// Initially no credentials
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials initially")
	}

	// Save credentials
	creds := AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	if err := mgr.SaveAWS(ctx, creds); err != nil {
		t.Errorf("SaveAWS failed: %v", err)
	}

	// Should have credentials now
	if !mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected AWS credentials after save")
	}

	// Load credentials
	loaded, err := mgr.LoadAWS(ctx)
	if err != nil {
		t.Errorf("LoadAWS failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadAWS returned nil")
	}
	if loaded.AccessKeyID != creds.AccessKeyID {
		t.Errorf("AccessKeyID mismatch: got %s, want %s", loaded.AccessKeyID, creds.AccessKeyID)
	}
	if loaded.SecretAccessKey != creds.SecretAccessKey {
		t.Errorf("SecretAccessKey mismatch: got %s, want %s", loaded.SecretAccessKey, creds.SecretAccessKey)
	}

	// Delete credentials
	if err := mgr.DeleteAWS(ctx); err != nil {
		t.Errorf("DeleteAWS failed: %v", err)
	}

	// Should have no credentials now
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials after delete")
	}
}

func TestManagerHetznerOperations(t *testing.T) {
	ctx := context.Background()
	mgr := NewMockManager()

	// Initially no credentials
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials initially")
	}

	// Save credentials
	creds := HetznerCredentials{
		APIToken: "test-api-token-12345",
	}
	if err := mgr.SaveHetzner(ctx, creds); err != nil {
		t.Errorf("SaveHetzner failed: %v", err)
	}

	// Should have credentials now
	if !mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected Hetzner credentials after save")
	}

	// Load credentials
	loaded, err := mgr.LoadHetzner(ctx)
	if err != nil {
		t.Errorf("LoadHetzner failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadHetzner returned nil")
	}
	if loaded.APIToken != creds.APIToken {
		t.Errorf("APIToken mismatch: got %s, want %s", loaded.APIToken, creds.APIToken)
	}

	// Delete credentials
	if err := mgr.DeleteHetzner(ctx); err != nil {
		t.Errorf("DeleteHetzner failed: %v", err)
	}

	// Should have no credentials now
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials after delete")
	}
}

func TestManagerErrorHandling(t *testing.T) {
	ctx := context.Background()
	mgr := NewMockManager()
	testErr := errors.New("test error")
	mgr.SetError(testErr)

	// All operations should return the error
	if err := mgr.SaveAWS(ctx, AWSCredentials{}); err != testErr {
		t.Errorf("SaveAWS error mismatch: got %v, want %v", err, testErr)
	}
	if _, err := mgr.LoadAWS(ctx); err != testErr {
		t.Errorf("LoadAWS error mismatch: got %v, want %v", err, testErr)
	}
	if err := mgr.DeleteAWS(ctx); err != testErr {
		t.Errorf("DeleteAWS error mismatch: got %v, want %v", err, testErr)
	}
	if err := mgr.SaveHetzner(ctx, HetznerCredentials{}); err != testErr {
		t.Errorf("SaveHetzner error mismatch: got %v, want %v", err, testErr)
	}
	if _, err := mgr.LoadHetzner(ctx); err != testErr {
		t.Errorf("LoadHetzner error mismatch: got %v, want %v", err, testErr)
	}
	if err := mgr.DeleteHetzner(ctx); err != testErr {
		t.Errorf("DeleteHetzner error mismatch: got %v, want %v", err, testErr)
	}
}

func TestHasCredentialsUnknownProvider(t *testing.T) {
	ctx := context.Background()
	mgr := NewMockManager()

	if mgr.HasCredentials(ctx, "unknown-provider") {
		t.Error("Expected false for unknown provider")
	}
}
