package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/gonfva/my-own-vpn/internal/app"
	"github.com/gonfva/my-own-vpn/internal/config"
	"github.com/gonfva/my-own-vpn/internal/ui"
)

// Version is set during build
var Version = "dev"

var (
	tray           *ui.TrayApp
	settingsWindow *ui.SettingsWindow
	controller     *app.Controller
	appConfig      *config.Config
)

func main() {
	fmt.Printf("My Own VPN v%s\n", Version)

	// Load configuration
	var err error
	appConfig, err = config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		appConfig = config.DefaultConfig()
	}

	// Create the controller
	controller = app.NewController()
	controller.SetOnStateChange(onControllerStateChange)
	controller.SetOnError(onControllerError)

	// Create the settings window (Fyne app will be created in RunFyneLoop)
	settingsWindow = ui.NewSettingsWindow()

	// Load config into settings window
	settingsWindow.LoadConfig(configToSettingsConfig(appConfig))
	settingsWindow.SetOnSave(onSettingsSaved)

	// Create the system tray application
	tray = ui.NewTrayApp()

	// Set up callbacks
	tray.SetCallbacks(
		onConnect,
		onDisconnect,
		onSettings,
		onQuit,
	)

	// Initialize the system tray menu after Fyne app has started.
	// The callback receives the Fyne app instance once it's created and the
	// event loop is running. This avoids "tray not ready" errors that occur
	// when the app is created too early before the event loop starts.
	settingsWindow.SetOnStarted(func(fyneApp fyne.App) {
		tray.SetFyneApp(fyneApp)
		tray.Setup()
	})

	// Run Fyne event loop on main goroutine (required by GLFW on Windows/macOS)
	// This blocks until Quit is called
	settingsWindow.RunFyneLoop()
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

func onSettingsSaved(settingsConfig ui.SettingsConfig) {
	fmt.Printf("Settings saved - Provider: %s, Region: %s\n", settingsConfig.Provider, settingsConfig.Region)

	// Extract non-sensitive config
	appConfig = settingsConfigToConfig(settingsConfig)

	// Save to disk
	if err := config.Save(appConfig); err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		tray.SetError(fmt.Sprintf("Failed to save settings: %v", err))
		return
	}

	// TODO: Store credentials in credential manager
	// TODO: Update controller/provider with new config

	fmt.Println("Configuration saved successfully")
}

func onQuit() {
	fmt.Println("Quit clicked - shutting down")
	// The Fyne app quit is handled by the tray's quit menu item callback
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

// configToSettingsConfig converts Config to SettingsConfig (merge with empty credentials)
func configToSettingsConfig(cfg *config.Config) ui.SettingsConfig {
	return ui.SettingsConfig{
		Provider:           cfg.Provider,
		Region:             cfg.Region,
		InstanceType:       cfg.InstanceType,
		IdleTimeoutEnabled: cfg.IdleTimeoutEnabled,
		IdleTimeoutMinutes: cfg.IdleTimeoutMinutes,
		// Credentials left empty - will be loaded from credential manager in future
	}
}

// settingsConfigToConfig extracts Config from SettingsConfig (strip credentials)
func settingsConfigToConfig(settings ui.SettingsConfig) *config.Config {
	return &config.Config{
		Provider:           settings.Provider,
		Region:             settings.Region,
		InstanceType:       settings.InstanceType,
		IdleTimeoutEnabled: settings.IdleTimeoutEnabled,
		IdleTimeoutMinutes: settings.IdleTimeoutMinutes,
	}
}
