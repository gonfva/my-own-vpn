//go:build !darwin && !windows

package credentials

// NewManager creates the appropriate credential manager for the current platform.
// On unsupported platforms, this returns ErrNoImplementation until platform-specific
// implementations are added (Linux Secret Service, etc.)
func NewManager() (Manager, error) {
	// TODO: Add Linux Secret Service implementation
	// TODO: Fall back to encrypted file storage

	return nil, ErrNoImplementation
}
