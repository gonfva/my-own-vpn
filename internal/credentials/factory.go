package credentials

import "errors"

// Error definitions for credential management
var (
	// ErrNoImplementation is returned when no credential manager implementation is available
	ErrNoImplementation = errors.New("no credential manager implementation available")
	// ErrNotFound is returned when credentials are not found in storage
	ErrNotFound = errors.New("credentials not found")
)
