//go:build darwin

package credentials

// NewManager creates the appropriate credential manager for macOS.
// It uses the macOS Keychain for secure credential storage.
func NewManager() (Manager, error) {
	return newKeychainManager()
}
