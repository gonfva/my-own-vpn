//go:build !darwin && !windows

package credentials

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	// On Linux and other platforms, NewManager returns a fallbackManager
	mgr, err := NewManager()

	if err != nil {
		t.Errorf("NewManager() failed: %v", err)
	}

	if mgr == nil {
		t.Error("Expected non-nil manager on Linux/other platforms")
	}
}
