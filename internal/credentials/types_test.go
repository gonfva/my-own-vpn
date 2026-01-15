package credentials

import "testing"

func TestAWSCredentialsIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		creds    *AWSCredentials
		expected bool
	}{
		{
			name:     "nil credentials",
			creds:    nil,
			expected: true,
		},
		{
			name:     "empty struct",
			creds:    &AWSCredentials{},
			expected: true,
		},
		{
			name: "only access key",
			creds: &AWSCredentials{
				AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
			},
			expected: false,
		},
		{
			name: "only secret key",
			creds: &AWSCredentials{
				SecretAccessKey: "secret",
			},
			expected: false,
		},
		{
			name: "both keys",
			creds: &AWSCredentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.creds.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHetznerCredentialsIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		creds    *HetznerCredentials
		expected bool
	}{
		{
			name:     "nil credentials",
			creds:    nil,
			expected: true,
		},
		{
			name:     "empty struct",
			creds:    &HetznerCredentials{},
			expected: true,
		},
		{
			name: "with token",
			creds: &HetznerCredentials{
				APIToken: "test-token",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.creds.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCredentialsStruct(t *testing.T) {
	// Test that Credentials struct properly holds both credential types
	creds := Credentials{
		AWS: &AWSCredentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
		Hetzner: &HetznerCredentials{
			APIToken: "test-token",
		},
	}

	if creds.AWS == nil {
		t.Error("Expected AWS credentials to be set")
	}
	if creds.Hetzner == nil {
		t.Error("Expected Hetzner credentials to be set")
	}

	if creds.AWS.AccessKeyID != "test-key" {
		t.Errorf("AccessKeyID mismatch: got %s, want test-key", creds.AWS.AccessKeyID)
	}
	if creds.Hetzner.APIToken != "test-token" {
		t.Errorf("APIToken mismatch: got %s, want test-token", creds.Hetzner.APIToken)
	}
}
