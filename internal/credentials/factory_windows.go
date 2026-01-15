//go:build windows

package credentials

// NewManager creates the appropriate credential manager for Windows.
// It uses the Windows Credential Manager for secure credential storage.
func NewManager() (Manager, error) {
	return newWincredManager()
}
