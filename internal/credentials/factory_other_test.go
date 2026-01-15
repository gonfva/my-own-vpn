//go:build !darwin

package credentials

import (
	"errors"
	"testing"
)

func TestNewManager(t *testing.T) {
	// On non-darwin platforms, NewManager returns ErrNoImplementation
	// until platform-specific implementations are added
	mgr, err := NewManager()

	if mgr != nil {
		t.Error("Expected nil manager on non-darwin platforms")
	}

	if !errors.Is(err, ErrNoImplementation) {
		t.Errorf("Expected ErrNoImplementation, got: %v", err)
	}
}
