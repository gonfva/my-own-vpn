//go:build !darwin && !windows

package credentials

// NewManager creates the appropriate credential manager for the current platform.
// On Linux and other platforms, this uses the encrypted file fallback storage.
// In the future, this could be enhanced to use Linux Secret Service when available.
func NewManager() (Manager, error) {
	// TODO: Add Linux Secret Service implementation and use it when available
	// For now, use encrypted file fallback
	return newFallbackManager()
}
