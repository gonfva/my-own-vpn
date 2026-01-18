//go:build windows && cgo

package ui

import (
	"os"
	"sync"
	"syscall"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole = kernel32.NewProc("AllocConsole")
	procFreeConsole  = kernel32.NewProc("FreeConsole")
	procGetConsole   = kernel32.NewProc("GetConsoleWindow")
	procSetStdHandle = kernel32.NewProc("SetStdHandle")
)

const (
	stdOutputHandle = ^uintptr(0) - 10 + 1 // STD_OUTPUT_HANDLE = -11
	stdErrorHandle  = ^uintptr(0) - 11 + 1 // STD_ERROR_HANDLE = -12
)

// windowsConsoleController implements ConsoleController for Windows.
type windowsConsoleController struct {
	mu      sync.Mutex
	visible bool
}

func newConsoleController() ConsoleController {
	return &windowsConsoleController{
		visible: false,
	}
}

// ShowConsole allocates a new console and redirects stdout/stderr to it.
func (c *windowsConsoleController) ShowConsole() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.visible {
		return true
	}

	// Allocate a new console
	ret, _, _ := procAllocConsole.Call()
	if ret == 0 {
		// AllocConsole failed, but we may already have a console
		// Check if we have a console window
		hwnd, _, _ := procGetConsole.Call()
		if hwnd == 0 {
			return false
		}
	}

	// Redirect stdout and stderr to the console
	if err := c.redirectStdHandles(); err != nil {
		return false
	}

	c.visible = true
	return true
}

// HideConsole frees the console window.
func (c *windowsConsoleController) HideConsole() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.visible {
		return true
	}

	// Free the console
	ret, _, _ := procFreeConsole.Call()
	if ret == 0 {
		return false
	}

	c.visible = false
	return true
}

// IsConsoleVisible returns whether the console is currently visible.
func (c *windowsConsoleController) IsConsoleVisible() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visible
}

// ToggleConsole toggles console visibility.
func (c *windowsConsoleController) ToggleConsole() bool {
	c.mu.Lock()
	visible := c.visible
	c.mu.Unlock()

	if visible {
		c.HideConsole()
		return false
	}
	c.ShowConsole()
	return true
}

// redirectStdHandles redirects os.Stdout and os.Stderr to the console.
func (c *windowsConsoleController) redirectStdHandles() error {
	// Open CONOUT$ for writing
	conout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	// Set the standard handles
	handle := conout.Fd()
	procSetStdHandle.Call(stdOutputHandle, handle)
	procSetStdHandle.Call(stdErrorHandle, handle)

	// Replace os.Stdout and os.Stderr
	os.Stdout = conout
	os.Stderr = conout

	return nil
}
