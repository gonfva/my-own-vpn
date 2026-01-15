package credentials

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Verify service name constants are set
	if ServiceName == "" {
		t.Error("ServiceName should not be empty")
	}
	if AWSAccount == "" {
		t.Error("AWSAccount should not be empty")
	}
	if HetznerAccount == "" {
		t.Error("HetznerAccount should not be empty")
	}

	// Verify provider constants
	if ProviderAWS == "" {
		t.Error("ProviderAWS should not be empty")
	}
	if ProviderHetzner == "" {
		t.Error("ProviderHetzner should not be empty")
	}
}

func TestErrorDefinitions(t *testing.T) {
	// Verify error types are defined
	if ErrNoImplementation == nil {
		t.Error("ErrNoImplementation should not be nil")
	}
	if ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}
}
