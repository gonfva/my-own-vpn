//go:build linux

package ui

// Platform-specific initialization for Linux
// The getlantern/systray library handles most Linux specifics automatically.
// This file can be extended for Linux-specific functionality.

func init() {
	// Linux-specific initialization can be added here
}

// IsMacOS returns false on Linux builds
func IsMacOS() bool {
	return false
}

// IsWindows returns false on Linux builds
func IsWindows() bool {
	return false
}
