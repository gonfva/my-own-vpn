//go:build !windows && cgo

package ui

// noopConsoleController is a no-op implementation for non-Windows platforms.
// On macOS and Linux, console management is not needed as the app runs
// from a terminal or has proper logging facilities.
type noopConsoleController struct{}

func newConsoleController() ConsoleController {
	return &noopConsoleController{}
}

// ShowConsole is a no-op on non-Windows platforms.
func (c *noopConsoleController) ShowConsole() bool {
	return false
}

// HideConsole is a no-op on non-Windows platforms.
func (c *noopConsoleController) HideConsole() bool {
	return false
}

// IsConsoleVisible always returns false on non-Windows platforms.
func (c *noopConsoleController) IsConsoleVisible() bool {
	return false
}

// ToggleConsole is a no-op on non-Windows platforms.
func (c *noopConsoleController) ToggleConsole() bool {
	return false
}

// IsWindows returns false on non-Windows platforms.
func IsWindows() bool {
	return false
}

// ConsoleSupportedMsg returns a message indicating console toggle is not supported.
func ConsoleSupportedMsg() string {
	return "Console toggle is only available on Windows"
}
