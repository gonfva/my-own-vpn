package ui

import (
	"testing"
)

func TestValidateConfig_ValidAWS(t *testing.T) {
	config := SettingsConfig{
		Provider:           ProviderAWS,
		Region:             "us-east-1",
		InstanceType:       "t3.micro",
		AWSAccessKey:       "AKIAIOSFODNN7EXAMPLE",
		AWSSecretKey:       "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		IdleTimeoutEnabled: false,
	}

	errors := ValidateConfig(config)

	if len(errors) > 0 {
		t.Errorf("expected no validation errors for valid AWS config, got: %v", errors)
	}
}

func TestValidateConfig_ValidHetzner(t *testing.T) {
	config := SettingsConfig{
		Provider:           ProviderHetzner,
		Region:             "nbg1",
		InstanceType:       "cx22",
		HetznerToken:       "abcdef123456",
		IdleTimeoutEnabled: false,
	}

	errors := ValidateConfig(config)

	if len(errors) > 0 {
		t.Errorf("expected no validation errors for valid Hetzner config, got: %v", errors)
	}
}

func TestValidateConfig_MissingProvider(t *testing.T) {
	config := SettingsConfig{
		Region: "us-east-1",
	}

	errors := ValidateConfig(config)

	found := false
	for _, err := range errors {
		if err == "Provider is required" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Provider is required' error")
	}
}

func TestValidateConfig_MissingRegion(t *testing.T) {
	config := SettingsConfig{
		Provider: ProviderAWS,
	}

	errors := ValidateConfig(config)

	found := false
	for _, err := range errors {
		if err == "Region is required" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Region is required' error")
	}
}

func TestValidateConfig_MissingAWSCredentials(t *testing.T) {
	config := SettingsConfig{
		Provider: ProviderAWS,
		Region:   "us-east-1",
	}

	errors := ValidateConfig(config)

	if len(errors) != 2 {
		t.Errorf("expected 2 validation errors, got %d: %v", len(errors), errors)
	}

	foundAccessKey := false
	foundSecretKey := false
	for _, err := range errors {
		if err == "AWS Access Key ID is required" {
			foundAccessKey = true
		}
		if err == "AWS Secret Access Key is required" {
			foundSecretKey = true
		}
	}

	if !foundAccessKey {
		t.Error("expected 'AWS Access Key ID is required' error")
	}
	if !foundSecretKey {
		t.Error("expected 'AWS Secret Access Key is required' error")
	}
}

func TestValidateConfig_MissingHetznerToken(t *testing.T) {
	config := SettingsConfig{
		Provider: ProviderHetzner,
		Region:   "nbg1",
	}

	errors := ValidateConfig(config)

	found := false
	for _, err := range errors {
		if err == "Hetzner API Token is required" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Hetzner API Token is required' error")
	}
}

func TestValidateConfig_InvalidIdleTimeout(t *testing.T) {
	config := SettingsConfig{
		Provider:           ProviderAWS,
		Region:             "us-east-1",
		AWSAccessKey:       "AKIAIOSFODNN7EXAMPLE",
		AWSSecretKey:       "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		IdleTimeoutEnabled: true,
		IdleTimeoutMinutes: 0,
	}

	errors := ValidateConfig(config)

	found := false
	for _, err := range errors {
		if err == "Idle timeout must be greater than 0 minutes" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Idle timeout must be greater than 0 minutes' error")
	}
}

func TestValidateConfig_ValidIdleTimeout(t *testing.T) {
	config := SettingsConfig{
		Provider:           ProviderAWS,
		Region:             "us-east-1",
		AWSAccessKey:       "AKIAIOSFODNN7EXAMPLE",
		AWSSecretKey:       "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		IdleTimeoutEnabled: true,
		IdleTimeoutMinutes: 30,
	}

	errors := ValidateConfig(config)

	if len(errors) > 0 {
		t.Errorf("expected no validation errors, got: %v", errors)
	}
}

func TestProviderConstants(t *testing.T) {
	if ProviderAWS != "AWS" {
		t.Errorf("expected ProviderAWS to be 'AWS', got %s", ProviderAWS)
	}

	if ProviderHetzner != "Hetzner" {
		t.Errorf("expected ProviderHetzner to be 'Hetzner', got %s", ProviderHetzner)
	}
}

func TestAWSRegions(t *testing.T) {
	if len(AWSRegions) == 0 {
		t.Error("AWSRegions should not be empty")
	}

	// Check that us-east-1 is included
	found := false
	for _, region := range AWSRegions {
		if region == "us-east-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AWSRegions should include us-east-1")
	}
}

func TestHetznerRegions(t *testing.T) {
	if len(HetznerRegions) == 0 {
		t.Error("HetznerRegions should not be empty")
	}

	// Check that nbg1 is included
	found := false
	for _, region := range HetznerRegions {
		if region == "nbg1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("HetznerRegions should include nbg1")
	}
}

func TestAWSInstanceTypes(t *testing.T) {
	if len(AWSInstanceTypes) == 0 {
		t.Error("AWSInstanceTypes should not be empty")
	}

	// Check that t3.micro is included
	found := false
	for _, instanceType := range AWSInstanceTypes {
		if instanceType == "t3.micro" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AWSInstanceTypes should include t3.micro")
	}
}

func TestHetznerInstanceTypes(t *testing.T) {
	if len(HetznerInstanceTypes) == 0 {
		t.Error("HetznerInstanceTypes should not be empty")
	}

	// Check that cx22 is included
	found := false
	for _, instanceType := range HetznerInstanceTypes {
		if instanceType == "cx22" {
			found = true
			break
		}
	}
	if !found {
		t.Error("HetznerInstanceTypes should include cx22")
	}
}

func TestSettingsConfigDefaults(t *testing.T) {
	// Test that zero-value config has expected defaults
	config := SettingsConfig{}

	if config.Provider != "" {
		t.Errorf("expected empty provider, got %s", config.Provider)
	}

	if config.IdleTimeoutEnabled != false {
		t.Error("expected idle timeout to be disabled by default")
	}

	if config.IdleTimeoutMinutes != 0 {
		t.Errorf("expected idle timeout minutes to be 0, got %d", config.IdleTimeoutMinutes)
	}
}

func TestDefaultSettingsConfig(t *testing.T) {
	config := DefaultSettingsConfig()

	if config.Provider != ProviderAWS {
		t.Errorf("expected default provider to be AWS, got %s", config.Provider)
	}

	if config.Region != "us-east-1" {
		t.Errorf("expected default region to be us-east-1, got %s", config.Region)
	}

	if config.InstanceType != "t3.micro" {
		t.Errorf("expected default instance type to be t3.micro, got %s", config.InstanceType)
	}

	if config.IdleTimeoutMinutes != 30 {
		t.Errorf("expected default idle timeout to be 30, got %d", config.IdleTimeoutMinutes)
	}
}
