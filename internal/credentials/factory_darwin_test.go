//go:build darwin

package credentials

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	// On darwin, NewManager returns a keychainManager
	mgr, err := NewManager()

	if err != nil {
		t.Errorf("NewManager() failed: %v", err)
	}

	if mgr == nil {
		t.Error("Expected non-nil manager on darwin")
	}
}
