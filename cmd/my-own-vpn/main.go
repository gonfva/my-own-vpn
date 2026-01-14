package main

import (
	"fmt"

	"github.com/gonfva/my-own-vpn/internal/ui"
)

// Version is set during build
var Version = "dev"

var (
	tray           *ui.TrayApp
	settingsWindow *ui.SettingsWindow
)

func main() {
	fmt.Printf("My Own VPN v%s\n", Version)

	// Create the settings window
	settingsWindow = ui.NewSettingsWindow()
	settingsWindow.SetOnSave(onSettingsSaved)

	// Start Fyne event loop in background goroutine
	go settingsWindow.RunFyneLoop()

	// Create the system tray application
	tray = ui.NewTrayApp()

	// Set up callbacks
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
	settingsWindow.Show()
}

func onSettingsSaved(config ui.SettingsConfig) {
	fmt.Printf("Settings saved - Provider: %s, Region: %s\n", config.Provider, config.Region)
	// TODO: Store credentials in credential manager
	// TODO: Update provider configuration
}

func onQuit() {
	fmt.Println("Quit clicked - shutting down")
	settingsWindow.StopFyneLoop()
}
