package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents non-sensitive application configuration
type Config struct {
	Provider           string `json:"provider"`             // "aws" or "hetzner"
	Region             string `json:"region"`               // Provider-specific region code
	InstanceType       string `json:"instance_type"`        // e.g., "t3.micro" or "cx22"
	IdleTimeoutEnabled bool   `json:"idle_timeout_enabled"` // Enable idle timeout
	IdleTimeoutMinutes int    `json:"idle_timeout_minutes"` // Timeout duration in minutes
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Provider:           "aws",
		Region:             "us-east-1",
		InstanceType:       "t3.micro",
		IdleTimeoutEnabled: false,
		IdleTimeoutMinutes: 30,
	}
}

// ConfigPath returns the OS-specific path to the configuration file
//
//nolint:revive // ConfigPath is intentionally named to be clear about returning a path
func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appDir := filepath.Join(configDir, "MyOwnVPN")
	return filepath.Join(appDir, "config.json"), nil
}

// Load loads configuration from disk, returning default config if file doesn't exist
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return loadFromPath(path)
}

// Save persists the configuration to disk with restrictive permissions
func Save(c *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return saveToPath(c, path)
}

// loadFromPath loads configuration from a specific path (used by Load and tests)
func loadFromPath(path string) (*Config, error) {
	// #nosec G304 -- Path is controlled by ConfigPath() using os.UserConfigDir()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// File doesn't exist - first run, return defaults
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file (corrupted): %w", err)
	}

	return &cfg, nil
}

// saveToPath saves configuration to a specific path (used by Save and tests)
func saveToPath(c *Config, path string) error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	// Validate before saving
	if err := Validate(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Ensure parent directory exists with restrictive permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON with indentation for readability
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration values are valid
func Validate(c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}

	// Normalize provider to lowercase for comparison
	provider := strings.ToLower(c.Provider)
	if provider != "aws" && provider != "hetzner" {
		return fmt.Errorf("invalid provider %q: must be aws or hetzner", c.Provider)
	}

	if c.Region == "" {
		return errors.New("region is required")
	}

	if c.InstanceType == "" {
		return errors.New("instance type is required")
	}

	// Only validate timeout if enabled
	if c.IdleTimeoutEnabled {
		if c.IdleTimeoutMinutes < 1 || c.IdleTimeoutMinutes > 1440 {
			return fmt.Errorf("idle timeout must be between 1 and 1440 minutes, got %d", c.IdleTimeoutMinutes)
		}
	}

	return nil
}
