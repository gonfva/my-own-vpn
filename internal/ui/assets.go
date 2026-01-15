package ui

import (
	"embed"

	"fyne.io/fyne/v2"
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

// IconDisconnectedResource returns the disconnected icon as a Fyne Resource
func IconDisconnectedResource() fyne.Resource {
	return fyne.NewStaticResource("disconnected.png", IconDisconnected())
}

// IconConnectingResource returns the connecting icon as a Fyne Resource
func IconConnectingResource() fyne.Resource {
	return fyne.NewStaticResource("connecting.png", IconConnecting())
}

// IconConnectedResource returns the connected icon as a Fyne Resource
func IconConnectedResource() fyne.Resource {
	return fyne.NewStaticResource("connected.png", IconConnected())
}

// IconErrorResource returns the error icon as a Fyne Resource
func IconErrorResource() fyne.Resource {
	return fyne.NewStaticResource("error.png", IconError())
}
