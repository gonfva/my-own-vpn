//go:build windows

package credentials

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	// On Windows, NewManager returns a wincredManager
	mgr, err := NewManager()

	if err != nil {
		t.Errorf("NewManager() failed: %v", err)
	}

	if mgr == nil {
		t.Error("Expected non-nil manager on Windows")
	}
}
