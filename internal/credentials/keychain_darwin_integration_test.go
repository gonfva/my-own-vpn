//go:build darwin && integration

package credentials

import (
	"context"
	"errors"
	"testing"
)

// Integration tests for keychainManager
// Run with: go test -tags=integration ./internal/credentials/...

func TestKeychainManagerAWSIntegration(t *testing.T) {
	mgr, err := newKeychainManager()
	if err != nil {
		t.Fatalf("failed to create keychain manager: %v", err)
	}

	ctx := context.Background()
	testCreds := AWSCredentials{
		AccessKeyID:     "test-key-id",
		SecretAccessKey: "test-secret-key",
	}

	// Clean up before test (ignore errors)
	_ = mgr.DeleteAWS(ctx)

	// Test that initially no credentials exist
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials initially")
	}

	// Test save
	if err := mgr.SaveAWS(ctx, testCreds); err != nil {
		t.Fatalf("SaveAWS failed: %v", err)
	}

	// Test HasCredentials returns true
	if !mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected HasCredentials to return true after save")
	}

	// Test load
	loaded, err := mgr.LoadAWS(ctx)
	if err != nil {
		t.Fatalf("LoadAWS failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadAWS returned nil credentials")
	}
	if loaded.AccessKeyID != testCreds.AccessKeyID {
		t.Errorf("AccessKeyID mismatch: got %s, want %s", loaded.AccessKeyID, testCreds.AccessKeyID)
	}
	if loaded.SecretAccessKey != testCreds.SecretAccessKey {
		t.Errorf("SecretAccessKey mismatch: got %s, want %s", loaded.SecretAccessKey, testCreds.SecretAccessKey)
	}

	// Test update (save again with different values)
	updatedCreds := AWSCredentials{
		AccessKeyID:     "updated-key-id",
		SecretAccessKey: "updated-secret-key",
	}
	if err := mgr.SaveAWS(ctx, updatedCreds); err != nil {
		t.Fatalf("SaveAWS (update) failed: %v", err)
	}

	loaded, err = mgr.LoadAWS(ctx)
	if err != nil {
		t.Fatalf("LoadAWS after update failed: %v", err)
	}
	if loaded.AccessKeyID != updatedCreds.AccessKeyID {
		t.Errorf("AccessKeyID after update mismatch: got %s, want %s", loaded.AccessKeyID, updatedCreds.AccessKeyID)
	}

	// Test delete
	if err := mgr.DeleteAWS(ctx); err != nil {
		t.Fatalf("DeleteAWS failed: %v", err)
	}

	// Verify credentials are gone
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials after delete")
	}

	// Test load returns ErrNotFound after delete
	_, err = mgr.LoadAWS(ctx)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}

	// Test delete of non-existent credentials doesn't error
	if err := mgr.DeleteAWS(ctx); err != nil {
		t.Errorf("DeleteAWS of non-existent credentials should not error: %v", err)
	}
}

func TestKeychainManagerHetznerIntegration(t *testing.T) {
	mgr, err := newKeychainManager()
	if err != nil {
		t.Fatalf("failed to create keychain manager: %v", err)
	}

	ctx := context.Background()
	testCreds := HetznerCredentials{
		APIToken: "test-api-token",
	}

	// Clean up before test (ignore errors)
	_ = mgr.DeleteHetzner(ctx)

	// Test that initially no credentials exist
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials initially")
	}

	// Test save
	if err := mgr.SaveHetzner(ctx, testCreds); err != nil {
		t.Fatalf("SaveHetzner failed: %v", err)
	}

	// Test HasCredentials returns true
	if !mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected HasCredentials to return true after save")
	}

	// Test load
	loaded, err := mgr.LoadHetzner(ctx)
	if err != nil {
		t.Fatalf("LoadHetzner failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadHetzner returned nil credentials")
	}
	if loaded.APIToken != testCreds.APIToken {
		t.Errorf("APIToken mismatch: got %s, want %s", loaded.APIToken, testCreds.APIToken)
	}

	// Test update
	updatedCreds := HetznerCredentials{
		APIToken: "updated-api-token",
	}
	if err := mgr.SaveHetzner(ctx, updatedCreds); err != nil {
		t.Fatalf("SaveHetzner (update) failed: %v", err)
	}

	loaded, err = mgr.LoadHetzner(ctx)
	if err != nil {
		t.Fatalf("LoadHetzner after update failed: %v", err)
	}
	if loaded.APIToken != updatedCreds.APIToken {
		t.Errorf("APIToken after update mismatch: got %s, want %s", loaded.APIToken, updatedCreds.APIToken)
	}

	// Test delete
	if err := mgr.DeleteHetzner(ctx); err != nil {
		t.Fatalf("DeleteHetzner failed: %v", err)
	}

	// Verify credentials are gone
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials after delete")
	}

	// Test load returns ErrNotFound after delete
	_, err = mgr.LoadHetzner(ctx)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestKeychainManagerHasCredentialsUnknownProvider(t *testing.T) {
	mgr, err := newKeychainManager()
	if err != nil {
		t.Fatalf("failed to create keychain manager: %v", err)
	}

	ctx := context.Background()

	if mgr.HasCredentials(ctx, "unknown-provider") {
		t.Error("Expected false for unknown provider")
	}
}
