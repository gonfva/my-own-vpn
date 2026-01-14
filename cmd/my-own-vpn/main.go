package main

import (
	"context"
	"fmt"

	"github.com/gonfva/my-own-vpn/internal/app"
	"github.com/gonfva/my-own-vpn/internal/ui"
)

// Version is set during build
var Version = "dev"

var (
	tray           *ui.TrayApp
	settingsWindow *ui.SettingsWindow
	controller     *app.Controller
)

func main() {
	fmt.Printf("My Own VPN v%s\n", Version)

	// Create the controller
	controller = app.NewController()
	controller.SetOnStateChange(onControllerStateChange)
	controller.SetOnError(onControllerError)

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
	go func() {
		ctx := context.Background()
		if err := controller.Connect(ctx); err != nil {
			tray.SetError(err.Error())
		}
	}()
}

func onDisconnect() {
	go func() {
		ctx := context.Background()
		if err := controller.Disconnect(ctx); err != nil {
			tray.SetError(err.Error())
		}
	}()
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

func onControllerStateChange(state app.State, message string) {
	fmt.Printf("State changed: %s - %s\n", state, message)

	switch state {
	case app.StateDisconnected:
		tray.UpdateStatus("Disconnected", false)

	case app.StateProvisioning, app.StateConnecting:
		tray.SetConnecting()

	case app.StateConnected:
		tray.UpdateStatus("Connected", true)

	case app.StateDisconnecting, app.StateDeprovisioning:
		tray.SetDisconnecting()

	case app.StateError:
		tray.SetError(message)
	}
}

func onControllerError(err error) {
	fmt.Printf("Controller error: %v\n", err)
}
