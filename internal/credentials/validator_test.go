package credentials

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestValidatorEmptyCredentials(t *testing.T) {
	validator := NewValidator()
	ctx := context.Background()

	// Test empty AWS credentials
	err := validator.ValidateAWS(ctx, AWSCredentials{})
	if err == nil {
		t.Error("expected error for empty AWS credentials")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}

	// Test empty Hetzner credentials
	err = validator.ValidateHetzner(ctx, HetznerCredentials{})
	if err == nil {
		t.Error("expected error for empty Hetzner credentials")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidatorInvalidAWSCredentials(t *testing.T) {
	// Skip if running in short mode (these tests make real API calls)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	validator := NewValidatorWithTimeout(10 * time.Second)
	ctx := context.Background()

	// Test with invalid credentials - should fail with invalid credentials error
	// Using obviously fake credentials that won't trigger secret scanners
	err := validator.ValidateAWS(ctx, AWSCredentials{
		AccessKeyID:     "INVALID_KEY_ID_FOR_TESTING",
		SecretAccessKey: "invalid_secret_key_for_testing_only",
	})

	if err == nil {
		t.Error("expected error for invalid AWS credentials")
	}
	// The error should be an invalid credentials error
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidatorInvalidHetznerCredentials(t *testing.T) {
	// Skip if running in short mode (these tests make real API calls)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	validator := NewValidatorWithTimeout(10 * time.Second)
	ctx := context.Background()

	// Test with invalid credentials - should fail with invalid credentials error
	err := validator.ValidateHetzner(ctx, HetznerCredentials{
		APIToken: "invalid-token-that-does-not-exist",
	})

	if err == nil {
		t.Error("expected error for invalid Hetzner credentials")
	}
	// The error should be an invalid credentials error
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidatorValidAWSCredentials(t *testing.T) {
	// Skip if no test credentials are provided
	accessKey := os.Getenv("TEST_AWS_ACCESS_KEY")
	secretKey := os.Getenv("TEST_AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("TEST_AWS_ACCESS_KEY and TEST_AWS_SECRET_KEY not set")
	}

	validator := NewValidator()
	ctx := context.Background()

	err := validator.ValidateAWS(ctx, AWSCredentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	if err != nil {
		t.Errorf("valid AWS credentials rejected: %v", err)
	}
}

func TestValidatorValidHetznerCredentials(t *testing.T) {
	// Skip if no test credentials are provided
	apiToken := os.Getenv("TEST_HETZNER_API_TOKEN")
	if apiToken == "" {
		t.Skip("TEST_HETZNER_API_TOKEN not set")
	}

	validator := NewValidator()
	ctx := context.Background()

	err := validator.ValidateHetzner(ctx, HetznerCredentials{
		APIToken: apiToken,
	})

	if err != nil {
		t.Errorf("valid Hetzner credentials rejected: %v", err)
	}
}

func TestValidatorTimeout(t *testing.T) {
	// Create validator with very short timeout
	validator := NewValidatorWithTimeout(1 * time.Nanosecond)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test AWS with cancelled context
	err := validator.ValidateAWS(ctx, AWSCredentials{
		AccessKeyID:     "test-key-id",
		SecretAccessKey: "test-secret-key",
	})
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	// Test Hetzner with cancelled context
	err = validator.ValidateHetzner(ctx, HetznerCredentials{
		APIToken: "some-token",
	})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestUserFriendlyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "invalid credentials",
			err:      ErrInvalidCredentials,
			expected: "The credentials are invalid. Please check and try again.",
		},
		{
			name:     "network error",
			err:      ErrNetworkError,
			expected: "Could not connect to the cloud provider. Please check your internet connection.",
		},
		{
			name:     "timeout error",
			err:      ErrValidationTimeout,
			expected: "Connection to the cloud provider timed out. Please try again.",
		},
		{
			name:     "wrapped invalid credentials",
			err:      errors.Join(ErrInvalidCredentials, errors.New("some details")),
			expected: "The credentials are invalid. Please check and try again.",
		},
		{
			name:     "unknown error",
			err:      errors.New("some unknown error"),
			expected: "Validation failed: some unknown error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := UserFriendlyError(tc.err)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
	if v.timeout != defaultValidationTimeout {
		t.Errorf("expected default timeout %v, got %v", defaultValidationTimeout, v.timeout)
	}
}

func TestNewValidatorWithTimeout(t *testing.T) {
	customTimeout := 5 * time.Second
	v := NewValidatorWithTimeout(customTimeout)
	if v == nil {
		t.Fatal("NewValidatorWithTimeout returned nil")
	}
	if v.timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, v.timeout)
	}
}
