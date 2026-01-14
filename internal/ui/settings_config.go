package ui

// Provider constants
const (
	ProviderAWS     = "AWS"
	ProviderHetzner = "Hetzner"
)

// AWSRegions contains available AWS regions for VPN deployment
var AWSRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-central-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"sa-east-1",
}

// HetznerRegions contains available Hetzner regions for VPN deployment
var HetznerRegions = []string{
	"nbg1",
	"fsn1",
	"hel1",
	"ash",
	"hil",
}

// AWSInstanceTypes contains available AWS instance types
var AWSInstanceTypes = []string{
	"t3.micro",
	"t3.small",
	"t3.medium",
}

// HetznerInstanceTypes contains available Hetzner instance types
var HetznerInstanceTypes = []string{
	"cx22",
	"cx32",
	"cx42",
}

// SettingsConfig holds all user-configurable settings
type SettingsConfig struct {
	Provider           string
	Region             string
	InstanceType       string
	AWSAccessKey       string
	AWSSecretKey       string
	HetznerToken       string
	IdleTimeoutEnabled bool
	IdleTimeoutMinutes int
}

// ValidateConfig validates the settings configuration
func ValidateConfig(config SettingsConfig) []string {
	var errors []string

	if config.Provider == "" {
		errors = append(errors, "Provider is required")
	}

	if config.Region == "" {
		errors = append(errors, "Region is required")
	}

	switch config.Provider {
	case ProviderAWS:
		if config.AWSAccessKey == "" {
			errors = append(errors, "AWS Access Key ID is required")
		}
		if config.AWSSecretKey == "" {
			errors = append(errors, "AWS Secret Access Key is required")
		}
	case ProviderHetzner:
		if config.HetznerToken == "" {
			errors = append(errors, "Hetzner API Token is required")
		}
	}

	if config.IdleTimeoutEnabled && config.IdleTimeoutMinutes <= 0 {
		errors = append(errors, "Idle timeout must be greater than 0 minutes")
	}

	return errors
}

// DefaultSettingsConfig returns the default settings configuration
func DefaultSettingsConfig() SettingsConfig {
	return SettingsConfig{
		Provider:           ProviderAWS,
		Region:             "us-east-1",
		InstanceType:       "t3.micro",
		IdleTimeoutMinutes: 30,
	}
}
