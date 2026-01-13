//go:build windows

package ui

// Platform-specific initialization for Windows
// The getlantern/systray library handles most Windows specifics automatically.
// This file can be extended for Windows-specific functionality such as:
// - Windows notification balloon tips
// - Custom icon handling (.ico format if needed)
// - Windows-specific registry integration

func init() {
	// Windows-specific initialization can be added here
}

// IsMacOS returns false on Windows builds
func IsMacOS() bool {
	return false
}

// IsWindows returns true on Windows builds
func IsWindows() bool {
	return true
}
