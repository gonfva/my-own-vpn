//go:build !windows && !darwin

package wireguard

import (
	"runtime"
)

// PlatformError indicates that the current platform is not supported.
type PlatformError struct {
	Platform string
}

func (e *PlatformError) Error() string {
	return "wireguard client not supported on platform: " + e.Platform
}

// NewClient returns an error on unsupported platforms.
func NewClient() (Client, error) {
	return nil, &PlatformError{Platform: runtime.GOOS}
}
