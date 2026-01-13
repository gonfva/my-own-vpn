package ui

import (
	"embed"
)

// Icon assets for system tray states
//
//go:embed assets/disconnected.png
//go:embed assets/connecting.png
//go:embed assets/connected.png
//go:embed assets/error.png
var iconFS embed.FS

// IconDisconnected returns the icon for disconnected state
func IconDisconnected() []byte {
	data, _ := iconFS.ReadFile("assets/disconnected.png")
	return data
}

// IconConnecting returns the icon for connecting state
func IconConnecting() []byte {
	data, _ := iconFS.ReadFile("assets/connecting.png")
	return data
}

// IconConnected returns the icon for connected state
func IconConnected() []byte {
	data, _ := iconFS.ReadFile("assets/connected.png")
	return data
}

// IconError returns the icon for error state
func IconError() []byte {
	data, _ := iconFS.ReadFile("assets/error.png")
	return data
}
