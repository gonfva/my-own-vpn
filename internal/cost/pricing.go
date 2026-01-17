package cost

// Pricing data for AWS and Hetzner cloud providers.
// These are approximate hourly rates and may change over time.
// Note: Costs are estimates and actual billing may vary.

// AWSPricing contains hourly rates (USD) for AWS EC2 instances by region.
// Rates are for on-demand Linux instances.
var AWSPricing = map[string]map[string]float64{
	"t3.nano": {
		"us-east-1":      0.0052,
		"us-east-2":      0.0052,
		"us-west-1":      0.0062,
		"us-west-2":      0.0052,
		"eu-west-1":      0.0059,
		"eu-west-2":      0.0062,
		"eu-central-1":   0.0060,
		"ap-northeast-1": 0.0068,
		"ap-southeast-1": 0.0059,
		"ap-southeast-2": 0.0066,
	},
	"t3.micro": {
		"us-east-1":      0.0104,
		"us-east-2":      0.0104,
		"us-west-1":      0.0124,
		"us-west-2":      0.0104,
		"eu-west-1":      0.0116,
		"eu-west-2":      0.0124,
		"eu-central-1":   0.0118,
		"ap-northeast-1": 0.0136,
		"ap-southeast-1": 0.0118,
		"ap-southeast-2": 0.0132,
	},
	"t3.small": {
		"us-east-1":      0.0208,
		"us-east-2":      0.0208,
		"us-west-1":      0.0248,
		"us-west-2":      0.0208,
		"eu-west-1":      0.0232,
		"eu-west-2":      0.0248,
		"eu-central-1":   0.0236,
		"ap-northeast-1": 0.0272,
		"ap-southeast-1": 0.0236,
		"ap-southeast-2": 0.0264,
	},
	"t3.medium": {
		"us-east-1":      0.0416,
		"us-east-2":      0.0416,
		"us-west-1":      0.0496,
		"us-west-2":      0.0416,
		"eu-west-1":      0.0464,
		"eu-west-2":      0.0496,
		"eu-central-1":   0.0472,
		"ap-northeast-1": 0.0544,
		"ap-southeast-1": 0.0472,
		"ap-southeast-2": 0.0528,
	},
	"t2.micro": {
		"us-east-1":      0.0116,
		"us-east-2":      0.0116,
		"us-west-1":      0.0140,
		"us-west-2":      0.0116,
		"eu-west-1":      0.0134,
		"eu-west-2":      0.0140,
		"eu-central-1":   0.0134,
		"ap-northeast-1": 0.0152,
		"ap-southeast-1": 0.0134,
		"ap-southeast-2": 0.0148,
	},
	"t2.small": {
		"us-east-1":      0.0230,
		"us-east-2":      0.0230,
		"us-west-1":      0.0276,
		"us-west-2":      0.0230,
		"eu-west-1":      0.0268,
		"eu-west-2":      0.0276,
		"eu-central-1":   0.0268,
		"ap-northeast-1": 0.0304,
		"ap-southeast-1": 0.0268,
		"ap-southeast-2": 0.0296,
	},
	"t2.medium": {
		"us-east-1":      0.0464,
		"us-east-2":      0.0464,
		"us-west-1":      0.0556,
		"us-west-2":      0.0464,
		"eu-west-1":      0.0536,
		"eu-west-2":      0.0556,
		"eu-central-1":   0.0536,
		"ap-northeast-1": 0.0608,
		"ap-southeast-1": 0.0536,
		"ap-southeast-2": 0.0592,
	},
}

// HetznerPricing contains hourly rates (USD) for Hetzner cloud servers.
// Rates are converted from EUR monthly pricing to hourly USD approximation.
// Hetzner pricing is the same across all locations.
var HetznerPricing = map[string]float64{
	// Shared vCPU (Intel)
	"cx11": 0.0050, // ~3.29 EUR/month
	"cx21": 0.0088, // ~5.83 EUR/month
	"cx31": 0.0159, // ~10.59 EUR/month
	"cx41": 0.0269, // ~17.49 EUR/month
	"cx51": 0.0508, // ~32.99 EUR/month

	// Shared vCPU (AMD, high-performance)
	"cpx11": 0.0066, // ~4.49 EUR/month
	"cpx21": 0.0119, // ~8.49 EUR/month
	"cpx31": 0.0208, // ~14.99 EUR/month
	"cpx41": 0.0357, // ~26.49 EUR/month
	"cpx51": 0.0700, // ~52.99 EUR/month

	// ARM (Ampere)
	"cax11": 0.0053, // ~3.79 EUR/month
	"cax21": 0.0092, // ~6.49 EUR/month
	"cax31": 0.0166, // ~11.99 EUR/month
	"cax41": 0.0285, // ~20.49 EUR/month
}

// DefaultAWSInstanceType is the default instance type for AWS.
const DefaultAWSInstanceType = "t3.micro"

// DefaultHetznerInstanceType is the default instance type for Hetzner.
const DefaultHetznerInstanceType = "cx11"

// DefaultAWSRate is the fallback hourly rate for unknown AWS instances.
const DefaultAWSRate = 0.0104

// DefaultHetznerRate is the fallback hourly rate for unknown Hetzner instances.
const DefaultHetznerRate = 0.0050

// GetAWSHourlyRate returns the hourly rate and currency for an AWS instance.
// Falls back to default rate if instance type or region is not found.
func GetAWSHourlyRate(instanceType, region string) (rate float64, currency string) {
	if instances, ok := AWSPricing[instanceType]; ok {
		if rate, ok := instances[region]; ok {
			return rate, "USD"
		}
		// Region not found, use us-east-1 rate if available
		if rate, ok := instances["us-east-1"]; ok {
			return rate, "USD"
		}
	}
	return DefaultAWSRate, "USD"
}

// GetHetznerHourlyRate returns the hourly rate and currency for a Hetzner server.
// Falls back to default rate if server type is not found.
func GetHetznerHourlyRate(serverType string) (rate float64, currency string) {
	if rate, ok := HetznerPricing[serverType]; ok {
		return rate, "USD"
	}
	return DefaultHetznerRate, "USD"
}

// GetHourlyRate returns the hourly rate and currency for a provider and instance type.
// providerName should be "aws" or "hetzner".
func GetHourlyRate(providerName, instanceType, region string) (rate float64, currency string) {
	switch providerName {
	case "aws":
		return GetAWSHourlyRate(instanceType, region)
	case "hetzner":
		return GetHetznerHourlyRate(instanceType)
	default:
		return 0, "USD"
	}
}

// ListAWSInstanceTypes returns a list of supported AWS instance types.
func ListAWSInstanceTypes() []string {
	types := make([]string, 0, len(AWSPricing))
	for t := range AWSPricing {
		types = append(types, t)
	}
	return types
}

// ListHetznerInstanceTypes returns a list of supported Hetzner server types.
func ListHetznerInstanceTypes() []string {
	types := make([]string, 0, len(HetznerPricing))
	for t := range HetznerPricing {
		types = append(types, t)
	}
	return types
}
