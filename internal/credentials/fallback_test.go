package credentials

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackEncryption(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create manager with test directory
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testData := []byte("test secret data")

	encrypted, err := mgr.encrypt(testData)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Encrypted data should be longer than original (nonce + overhead)
	if len(encrypted) <= len(testData) {
		t.Error("encrypted data should be longer than original")
	}

	decrypted, err := mgr.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Error("decrypted data does not match original")
	}
}

func TestFallbackEncryptionDifferentNonces(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testData := []byte("test secret data")

	encrypted1, _ := mgr.encrypt(testData)
	encrypted2, _ := mgr.encrypt(testData)

	// Same data should produce different ciphertext due to random nonces
	if string(encrypted1) == string(encrypted2) {
		t.Error("encrypting same data twice should produce different ciphertext")
	}
}

func TestFallbackDecryptInvalidData(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test with too-short data
	_, err = mgr.decrypt([]byte("short"))
	if !errors.Is(err, ErrInvalidData) {
		t.Errorf("expected ErrInvalidData for short data, got: %v", err)
	}

	// Test with corrupted data
	encrypted, _ := mgr.encrypt([]byte("test"))
	encrypted[len(encrypted)-1] ^= 0xFF // Corrupt last byte
	_, err = mgr.decrypt(encrypted)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed for corrupted data, got: %v", err)
	}
}

func TestFallbackManagerAWS(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	testCreds := AWSCredentials{
		AccessKeyID:     "test-key-id",
		SecretAccessKey: "test-secret-key",
	}

	// Initially no credentials
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials initially")
	}

	// Test save
	if err := mgr.SaveAWS(ctx, testCreds); err != nil {
		t.Fatalf("SaveAWS failed: %v", err)
	}

	// Test HasCredentials
	if !mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected HasCredentials to return true after save")
	}

	// Test load
	loaded, err := mgr.LoadAWS(ctx)
	if err != nil {
		t.Fatalf("LoadAWS failed: %v", err)
	}
	if loaded.AccessKeyID != testCreds.AccessKeyID {
		t.Errorf("AccessKeyID mismatch: got %s, want %s", loaded.AccessKeyID, testCreds.AccessKeyID)
	}
	if loaded.SecretAccessKey != testCreds.SecretAccessKey {
		t.Errorf("SecretAccessKey mismatch: got %s, want %s", loaded.SecretAccessKey, testCreds.SecretAccessKey)
	}

	// Test update
	updatedCreds := AWSCredentials{
		AccessKeyID:     "updated-key-id",
		SecretAccessKey: "updated-secret-key",
	}
	if err := mgr.SaveAWS(ctx, updatedCreds); err != nil {
		t.Fatalf("SaveAWS (update) failed: %v", err)
	}

	loaded, _ = mgr.LoadAWS(ctx)
	if loaded.AccessKeyID != updatedCreds.AccessKeyID {
		t.Errorf("AccessKeyID after update mismatch: got %s, want %s", loaded.AccessKeyID, updatedCreds.AccessKeyID)
	}

	// Test delete
	if err := mgr.DeleteAWS(ctx); err != nil {
		t.Fatalf("DeleteAWS failed: %v", err)
	}

	// Verify deleted
	if mgr.HasCredentials(ctx, ProviderAWS) {
		t.Error("Expected no AWS credentials after delete")
	}

	_, err = mgr.LoadAWS(ctx)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestFallbackManagerHetzner(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	testCreds := HetznerCredentials{
		APIToken: "test-api-token",
	}

	// Initially no credentials
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials initially")
	}

	// Test save
	if err := mgr.SaveHetzner(ctx, testCreds); err != nil {
		t.Fatalf("SaveHetzner failed: %v", err)
	}

	// Test HasCredentials
	if !mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected HasCredentials to return true after save")
	}

	// Test load
	loaded, err := mgr.LoadHetzner(ctx)
	if err != nil {
		t.Fatalf("LoadHetzner failed: %v", err)
	}
	if loaded.APIToken != testCreds.APIToken {
		t.Errorf("APIToken mismatch: got %s, want %s", loaded.APIToken, testCreds.APIToken)
	}

	// Test delete
	if err := mgr.DeleteHetzner(ctx); err != nil {
		t.Fatalf("DeleteHetzner failed: %v", err)
	}

	// Verify deleted
	if mgr.HasCredentials(ctx, ProviderHetzner) {
		t.Error("Expected no Hetzner credentials after delete")
	}

	_, err = mgr.LoadHetzner(ctx)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestFallbackManagerUnknownProvider(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	if mgr.HasCredentials(ctx, "unknown-provider") {
		t.Error("Expected false for unknown provider")
	}
}

func TestFallbackManagerPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create first manager and save credentials
	mgr1, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create first manager: %v", err)
	}

	ctx := context.Background()
	testCreds := AWSCredentials{
		AccessKeyID:     "persistent-key-id",
		SecretAccessKey: "persistent-secret-key",
	}

	if err := mgr1.SaveAWS(ctx, testCreds); err != nil {
		t.Fatalf("SaveAWS failed: %v", err)
	}

	// Create second manager (simulating app restart) and load credentials
	mgr2, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}

	loaded, err := mgr2.LoadAWS(ctx)
	if err != nil {
		t.Fatalf("LoadAWS failed after 'restart': %v", err)
	}

	if loaded.AccessKeyID != testCreds.AccessKeyID {
		t.Errorf("Credentials not persisted correctly: got %s, want %s", loaded.AccessKeyID, testCreds.AccessKeyID)
	}
}

func TestFallbackManagerFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := createTestFallbackManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	if err := mgr.SaveAWS(ctx, AWSCredentials{AccessKeyID: "test", SecretAccessKey: "test"}); err != nil {
		t.Fatalf("SaveAWS failed: %v", err)
	}

	// Check credentials file permissions
	info, err := os.Stat(mgr.path)
	if err != nil {
		t.Fatalf("failed to stat credentials file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("credentials file has wrong permissions: got %o, want 0o600", perm)
	}

	// Check salt file permissions
	saltPath := filepath.Join(tempDir, saltFileName)
	info, err = os.Stat(saltPath)
	if err != nil {
		t.Fatalf("failed to stat salt file: %v", err)
	}

	perm = info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("salt file has wrong permissions: got %o, want 0o600", perm)
	}
}

// createTestFallbackManager creates a fallbackManager using a test directory
func createTestFallbackManager(tempDir string) (*fallbackManager, error) {
	key, err := deriveKey(tempDir)
	if err != nil {
		return nil, err
	}

	credPath := filepath.Join(tempDir, credentialsFileName)
	return &fallbackManager{key: key, path: credPath}, nil
}
