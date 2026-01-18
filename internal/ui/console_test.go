//go:build cgo

package ui

import (
	"runtime"
	"testing"
)

func TestConsoleControllerExists(t *testing.T) {
	if Console == nil {
		t.Fatal("Console controller is nil")
	}
}

func TestIsWindows(t *testing.T) {
	isWin := IsWindows()
	actuallyWindows := runtime.GOOS == "windows"

	if isWin != actuallyWindows {
		t.Errorf("IsWindows() = %v, but runtime.GOOS = %v", isWin, runtime.GOOS)
	}
}

func TestConsoleSupportedMsg(t *testing.T) {
	msg := ConsoleSupportedMsg()

	if runtime.GOOS == "windows" {
		if msg != "" {
			t.Errorf("expected empty message on Windows, got %q", msg)
		}
	} else {
		if msg == "" {
			t.Error("expected non-empty message on non-Windows platforms")
		}
	}
}

func TestConsoleOperations(t *testing.T) {
	// Initial state should be not visible
	if Console.IsConsoleVisible() {
		t.Error("expected console to not be visible initially")
	}

	if runtime.GOOS != "windows" {
		// On non-Windows platforms, all operations should return false
		if Console.ShowConsole() {
			t.Error("ShowConsole should return false on non-Windows")
		}
		if Console.HideConsole() {
			// HideConsole returns true when already hidden on Windows implementation
			// but for noop, it returns false
		}
		if Console.ToggleConsole() {
			t.Error("ToggleConsole should return false on non-Windows")
		}
		if Console.IsConsoleVisible() {
			t.Error("IsConsoleVisible should return false on non-Windows")
		}
	}
}

func TestToggleConsole(t *testing.T) {
	if runtime.GOOS != "windows" {
		// On non-Windows, toggle should always return false
		result := Console.ToggleConsole()
		if result {
			t.Error("ToggleConsole should return false on non-Windows")
		}
		if Console.IsConsoleVisible() {
			t.Error("console should never be visible on non-Windows")
		}
	}
	// Note: Windows-specific tests would require actual Windows environment
	// and might interfere with the test runner's console
}
