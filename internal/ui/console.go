//go:build cgo

package ui

// ConsoleController provides methods to show/hide the console window.
// This is primarily useful on Windows where GUI applications don't have
// a console by default, but users may want to see logs for debugging.
type ConsoleController interface {
	// ShowConsole shows the console window if hidden.
	// Returns true if the console is now visible.
	ShowConsole() bool

	// HideConsole hides the console window if visible.
	// Returns true if the console is now hidden.
	HideConsole() bool

	// IsConsoleVisible returns whether the console is currently visible.
	IsConsoleVisible() bool

	// ToggleConsole toggles the console visibility.
	// Returns true if the console is now visible.
	ToggleConsole() bool
}

// Console is the global console controller instance.
// It is initialized with platform-specific implementation.
var Console ConsoleController

func init() {
	Console = newConsoleController()
}
