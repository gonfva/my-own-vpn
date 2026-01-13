package main

import (
	"fmt"

	"github.com/gonfva/my-own-vpn/internal/ui"
)

// Version is set during build
var Version = "dev"

func main() {
	fmt.Printf("My Own VPN v%s\n", Version)

	// Create the system tray application
	tray := ui.NewTrayApp()

	// Set up callbacks with placeholder implementations
	tray.SetCallbacks(
		onConnect,
		onDisconnect,
		onSettings,
		onQuit,
	)

	// Run the system tray (blocks until quit)
	tray.Run()
}

func onConnect() {
	fmt.Println("Connect clicked - not yet implemented")
}

func onDisconnect() {
	fmt.Println("Disconnect clicked - not yet implemented")
}

func onSettings() {
	fmt.Println("Settings clicked - not yet implemented")
}

func onQuit() {
	fmt.Println("Quit clicked - shutting down")
}
