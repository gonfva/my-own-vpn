package credentials

import (
	"errors"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Currently NewManager returns ErrNoImplementation as platform-specific
	// implementations are in separate tickets
	mgr, err := NewManager()

	if mgr != nil {
		t.Error("Expected nil manager until implementations are added")
	}

	if !errors.Is(err, ErrNoImplementation) {
		t.Errorf("Expected ErrNoImplementation, got: %v", err)
	}
}

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
