//go:build darwin

package ui

// Platform-specific initialization for macOS
// The getlantern/systray library handles most macOS specifics automatically.
// This file can be extended for macOS-specific functionality such as:
// - Menu bar template image handling
// - Dark mode adaptation
// - macOS notification integration

func init() {
	// macOS-specific initialization can be added here
	// For example, setting up template images for proper dark/light mode support
}

// IsMacOS returns true on macOS builds
func IsMacOS() bool {
	return true
}

// IsWindows returns false on macOS builds
func IsWindows() bool {
	return false
}
