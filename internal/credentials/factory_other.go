//go:build !darwin

package credentials

// NewManager creates the appropriate credential manager for the current platform.
// On non-macOS platforms, this returns ErrNoImplementation until platform-specific
// implementations are added (Windows Credential Manager, Linux Secret Service, etc.)
func NewManager() (Manager, error) {
	// TODO: Add Windows Credential Manager implementation
	// TODO: Add Linux Secret Service implementation
	// TODO: Fall back to encrypted file storage

	return nil, ErrNoImplementation
}
