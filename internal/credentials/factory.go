package credentials

import "errors"

// ErrNoImplementation is returned when no credential manager implementation is available
var ErrNoImplementation = errors.New("no credential manager implementation available")

// NewManager creates the appropriate credential manager for the current platform.
// It tries to use the system keychain first, then falls back to encrypted file storage.
// Platform-specific implementations will be added in future tickets.
func NewManager() (Manager, error) {
	// TODO: Try keychain first (platform-specific implementations in future tickets)
	// if mgr, err := newKeychainManager(); err == nil {
	//     return mgr, nil
	// }

	// TODO: Fall back to encrypted file storage (future ticket)
	// return newFallbackManager()

	// For now, return error as implementations are in separate tickets
	return nil, ErrNoImplementation
}
