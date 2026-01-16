package credentials

import "errors"

// Error definitions for credential management
var (
	// ErrNoImplementation is returned when no credential manager implementation is available
	ErrNoImplementation = errors.New("no credential manager implementation available")
	// ErrNotFound is returned when credentials are not found in storage
	ErrNotFound = errors.New("credentials not found")
	// ErrInvalidData is returned when encrypted data is malformed
	ErrInvalidData = errors.New("invalid encrypted data")
	// ErrDecryptionFailed is returned when decryption fails (wrong key or corrupted data)
	ErrDecryptionFailed = errors.New("decryption failed")
)
