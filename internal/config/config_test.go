package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.Provider != "aws" {
		t.Errorf("expected default provider aws, got %s", cfg.Provider)
	}

	if cfg.Region != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %s", cfg.Region)
	}

	if cfg.InstanceType != "t3.micro" {
		t.Errorf("expected default instance type t3.micro, got %s", cfg.InstanceType)
	}

	if cfg.IdleTimeoutEnabled {
		t.Error("expected idle timeout disabled by default")
	}

	if cfg.IdleTimeoutMinutes != 30 {
		t.Errorf("expected default idle timeout 30 minutes, got %d", cfg.IdleTimeoutMinutes)
	}

	// Default config should pass validation
	if err := Validate(cfg); err != nil {
		t.Errorf("default config should be valid, got error: %v", err)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath returned error: %v", err)
	}

	if path == "" {
		t.Fatal("ConfigPath returned empty string")
	}

	if !strings.Contains(path, "MyOwnVPN") {
		t.Errorf("path should contain MyOwnVPN, got %s", path)
	}

	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("path should end with config.json, got %s", path)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("path should be absolute, got %s", path)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorText string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorText: "nil",
		},
		{
			name:      "valid aws config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "valid hetzner config",
			config: &Config{
				Provider:           "hetzner",
				Region:             "nbg1",
				InstanceType:       "cx22",
				IdleTimeoutEnabled: false,
				IdleTimeoutMinutes: 30,
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider:     "invalid",
				Region:       "us-east-1",
				InstanceType: "t3.micro",
			},
			wantError: true,
			errorText: "provider",
		},
		{
			name: "empty region",
			config: &Config{
				Provider:     "aws",
				Region:       "",
				InstanceType: "t3.micro",
			},
			wantError: true,
			errorText: "region",
		},
		{
			name: "empty instance type",
			config: &Config{
				Provider:     "aws",
				Region:       "us-east-1",
				InstanceType: "",
			},
			wantError: true,
			errorText: "instance type",
		},
		{
			name: "timeout too low",
			config: &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: true,
				IdleTimeoutMinutes: 0,
			},
			wantError: true,
			errorText: "timeout",
		},
		{
			name: "timeout too high",
			config: &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: true,
				IdleTimeoutMinutes: 1441,
			},
			wantError: true,
			errorText: "timeout",
		},
		{
			name: "timeout valid at boundary - minimum",
			config: &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: true,
				IdleTimeoutMinutes: 1,
			},
			wantError: false,
		},
		{
			name: "timeout valid at boundary - maximum",
			config: &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: true,
				IdleTimeoutMinutes: 1440,
			},
			wantError: false,
		},
		{
			name: "timeout not validated when disabled",
			config: &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: false,
				IdleTimeoutMinutes: 9999, // Invalid but shouldn't matter
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorText != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorText)) {
					t.Errorf("expected error containing %q, got %q", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Create a custom config
	cfg := &Config{
		Provider:           "hetzner",
		Region:             "nbg1",
		InstanceType:       "cx22",
		IdleTimeoutEnabled: true,
		IdleTimeoutMinutes: 60,
	}

	// Save the config
	if err := saveToPath(cfg, testPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load the config back
	loaded, err := loadFromPath(testPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify all fields match
	if loaded.Provider != cfg.Provider {
		t.Errorf("provider mismatch: expected %s, got %s", cfg.Provider, loaded.Provider)
	}
	if loaded.Region != cfg.Region {
		t.Errorf("region mismatch: expected %s, got %s", cfg.Region, loaded.Region)
	}
	if loaded.InstanceType != cfg.InstanceType {
		t.Errorf("instance type mismatch: expected %s, got %s", cfg.InstanceType, loaded.InstanceType)
	}
	if loaded.IdleTimeoutEnabled != cfg.IdleTimeoutEnabled {
		t.Errorf("idle timeout enabled mismatch: expected %v, got %v", cfg.IdleTimeoutEnabled, loaded.IdleTimeoutEnabled)
	}
	if loaded.IdleTimeoutMinutes != cfg.IdleTimeoutMinutes {
		t.Errorf("idle timeout minutes mismatch: expected %d, got %d", cfg.IdleTimeoutMinutes, loaded.IdleTimeoutMinutes)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Use temp directory that doesn't have a config file
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "nonexistent", "config.json")

	// Load should return default config without error
	cfg, err := loadFromPath(testPath)
	if err != nil {
		t.Fatalf("Load should not return error for non-existent file, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Verify it's the default config
	defaultCfg := DefaultConfig()
	if cfg.Provider != defaultCfg.Provider {
		t.Errorf("expected default provider %s, got %s", defaultCfg.Provider, cfg.Provider)
	}
	if cfg.Region != defaultCfg.Region {
		t.Errorf("expected default region %s, got %s", defaultCfg.Region, cfg.Region)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Write invalid JSON to file
	invalidJSON := []byte("{invalid json content")
	if err := os.WriteFile(testPath, invalidJSON, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load should return error
	_, err := loadFromPath(testPath)
	if err == nil {
		t.Fatal("Load should return error for corrupted file")
	}

	// Error should mention corruption
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption, got: %v", err)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "subdir", "nested", "config.json")

	// Directory should not exist yet
	parentDir := filepath.Dir(testPath)
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Fatal("parent directory should not exist yet")
	}

	// Save should create directories
	cfg := DefaultConfig()
	if err := saveToPath(cfg, testPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}

	// Verify directory permissions (0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("directory permissions should be 0700, got %o", info.Mode().Perm())
	}
}

func TestSaveFilePermissions(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Save config
	cfg := DefaultConfig()
	if err := saveToPath(cfg, testPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	// Verify file has 0600 permissions
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions should be 0600, got %o", info.Mode().Perm())
	}
}

func TestConcurrentSaves(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Launch multiple goroutines saving different configs
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			cfg := &Config{
				Provider:           "aws",
				Region:             "us-east-1",
				InstanceType:       "t3.micro",
				IdleTimeoutEnabled: false,
				IdleTimeoutMinutes: 30 + index, // Make each save slightly different
			}

			if err := saveToPath(cfg, testPath); err != nil {
				t.Errorf("concurrent save %d failed: %v", index, err)
			}
		}(i)
	}

	// Wait for all saves to complete
	wg.Wait()

	// Verify file is still valid (can be loaded without error)
	cfg, err := loadFromPath(testPath)
	if err != nil {
		t.Fatalf("Load after concurrent saves failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load returned nil after concurrent saves")
	}

	// Verify it's a valid config
	if err := Validate(cfg); err != nil {
		t.Errorf("config invalid after concurrent saves: %v", err)
	}
}

func TestSaveNilConfig(t *testing.T) {
	err := Save(nil)
	if err == nil {
		t.Error("Save should return error for nil config")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil, got: %v", err)
	}
}

func TestSaveInvalidConfig(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Try to save invalid config
	cfg := &Config{
		Provider:     "invalid-provider",
		Region:       "us-east-1",
		InstanceType: "t3.micro",
	}

	err := saveToPath(cfg, testPath)
	if err == nil {
		t.Error("Save should return error for invalid config")
	}

	// File should not be created
	if _, err := os.Stat(testPath); !os.IsNotExist(err) {
		t.Error("file should not be created for invalid config")
	}
}
