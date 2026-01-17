package cost

import (
	"testing"
)

func TestGetAWSHourlyRate(t *testing.T) {
	tests := []struct {
		instanceType string
		region       string
		expectedRate float64
	}{
		{"t3.micro", "us-east-1", 0.0104},
		{"t3.micro", "eu-west-1", 0.0116},
		{"t3.small", "us-east-1", 0.0208},
		{"t3.nano", "us-west-2", 0.0052},
		{"t2.micro", "us-east-1", 0.0116},
	}

	for _, tc := range tests {
		t.Run(tc.instanceType+"_"+tc.region, func(t *testing.T) {
			rate, currency := GetAWSHourlyRate(tc.instanceType, tc.region)
			if rate != tc.expectedRate {
				t.Errorf("expected rate %f, got %f", tc.expectedRate, rate)
			}
			if currency != "USD" {
				t.Errorf("expected USD, got %s", currency)
			}
		})
	}
}

func TestGetAWSHourlyRateFallback(t *testing.T) {
	// Unknown instance type should fall back to default
	rate, currency := GetAWSHourlyRate("unknown-type", "us-east-1")
	if rate != DefaultAWSRate {
		t.Errorf("expected default rate %f, got %f", DefaultAWSRate, rate)
	}
	if currency != "USD" {
		t.Errorf("expected USD, got %s", currency)
	}

	// Unknown region should fall back to us-east-1 rate
	rate, currency = GetAWSHourlyRate("t3.micro", "unknown-region")
	if rate != 0.0104 {
		t.Errorf("expected us-east-1 fallback rate 0.0104, got %f", rate)
	}
	if currency != "USD" {
		t.Errorf("expected USD, got %s", currency)
	}
}

func TestGetHetznerHourlyRate(t *testing.T) {
	tests := []struct {
		serverType   string
		expectedRate float64
	}{
		{"cx11", 0.0050},
		{"cx21", 0.0088},
		{"cx31", 0.0159},
		{"cpx11", 0.0066},
		{"cax11", 0.0053},
	}

	for _, tc := range tests {
		t.Run(tc.serverType, func(t *testing.T) {
			rate, currency := GetHetznerHourlyRate(tc.serverType)
			if rate != tc.expectedRate {
				t.Errorf("expected rate %f, got %f", tc.expectedRate, rate)
			}
			if currency != "USD" {
				t.Errorf("expected USD, got %s", currency)
			}
		})
	}
}

func TestGetHetznerHourlyRateFallback(t *testing.T) {
	// Unknown server type should fall back to default
	rate, currency := GetHetznerHourlyRate("unknown-type")
	if rate != DefaultHetznerRate {
		t.Errorf("expected default rate %f, got %f", DefaultHetznerRate, rate)
	}
	if currency != "USD" {
		t.Errorf("expected USD, got %s", currency)
	}
}

func TestGetHourlyRate(t *testing.T) {
	tests := []struct {
		provider     string
		instanceType string
		region       string
		expectedRate float64
		expectedCurr string
	}{
		{"aws", "t3.micro", "us-east-1", 0.0104, "USD"},
		{"aws", "t3.small", "eu-west-1", 0.0232, "USD"},
		{"hetzner", "cx11", "", 0.0050, "USD"},
		{"hetzner", "cpx11", "fsn1", 0.0066, "USD"},
		{"unknown", "any", "any", 0, "USD"},
	}

	for _, tc := range tests {
		t.Run(tc.provider+"_"+tc.instanceType, func(t *testing.T) {
			rate, currency := GetHourlyRate(tc.provider, tc.instanceType, tc.region)
			if rate != tc.expectedRate {
				t.Errorf("expected rate %f, got %f", tc.expectedRate, rate)
			}
			if currency != tc.expectedCurr {
				t.Errorf("expected %s, got %s", tc.expectedCurr, currency)
			}
		})
	}
}

func TestListAWSInstanceTypes(t *testing.T) {
	types := ListAWSInstanceTypes()
	if len(types) == 0 {
		t.Error("expected non-empty list of AWS instance types")
	}

	// Verify known types are in the list
	typeMap := make(map[string]bool)
	for _, typ := range types {
		typeMap[typ] = true
	}

	expected := []string{"t3.micro", "t3.small", "t3.nano"}
	for _, exp := range expected {
		if !typeMap[exp] {
			t.Errorf("expected %s in list", exp)
		}
	}
}

func TestListHetznerInstanceTypes(t *testing.T) {
	types := ListHetznerInstanceTypes()
	if len(types) == 0 {
		t.Error("expected non-empty list of Hetzner server types")
	}

	// Verify known types are in the list
	typeMap := make(map[string]bool)
	for _, typ := range types {
		typeMap[typ] = true
	}

	expected := []string{"cx11", "cx21", "cpx11", "cax11"}
	for _, exp := range expected {
		if !typeMap[exp] {
			t.Errorf("expected %s in list", exp)
		}
	}
}

func TestAWSPricingDataExists(t *testing.T) {
	// Verify pricing data exists for default instance type
	if _, ok := AWSPricing[DefaultAWSInstanceType]; !ok {
		t.Errorf("pricing data missing for default AWS instance type %s", DefaultAWSInstanceType)
	}

	// Verify at least us-east-1 region exists for each instance type
	for instanceType, regions := range AWSPricing {
		if _, ok := regions["us-east-1"]; !ok {
			t.Errorf("us-east-1 pricing missing for %s", instanceType)
		}
	}
}

func TestHetznerPricingDataExists(t *testing.T) {
	// Verify pricing data exists for default server type
	if _, ok := HetznerPricing[DefaultHetznerInstanceType]; !ok {
		t.Errorf("pricing data missing for default Hetzner server type %s", DefaultHetznerInstanceType)
	}
}

func TestPricingRatesArePositive(t *testing.T) {
	// All AWS rates should be positive
	for instanceType, regions := range AWSPricing {
		for region, rate := range regions {
			if rate <= 0 {
				t.Errorf("AWS %s in %s has non-positive rate: %f", instanceType, region, rate)
			}
		}
	}

	// All Hetzner rates should be positive
	for serverType, rate := range HetznerPricing {
		if rate <= 0 {
			t.Errorf("Hetzner %s has non-positive rate: %f", serverType, rate)
		}
	}
}
