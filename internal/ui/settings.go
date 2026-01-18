//go:build cgo

package ui

import (
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// SettingsWindow manages the settings window UI
type SettingsWindow struct {
	mu sync.Mutex

	fyneApp fyne.App
	window  fyne.Window
	visible bool

	// Callbacks
	onSave    func(config SettingsConfig)
	onStarted func(fyne.App) // Callback to be executed when the app has started

	// Provider selection
	providerSelect *widget.Select

	// Region and instance type
	regionSelect       *widget.Select
	instanceTypeSelect *widget.Select

	// AWS credential fields
	awsAccessKeyEntry *widget.Entry
	awsSecretKeyEntry *widget.Entry
	awsContainer      *fyne.Container

	// Hetzner credential fields
	hetznerTokenEntry *widget.Entry
	hetznerContainer  *fyne.Container

	// Preferences
	idleTimeoutCheck *widget.Check
	idleTimeoutEntry *widget.Entry

	// Current config
	currentConfig SettingsConfig
}

// NewSettingsWindow creates a new SettingsWindow instance
func NewSettingsWindow() *SettingsWindow {
	s := &SettingsWindow{
		currentConfig: SettingsConfig{
			Provider:           ProviderAWS,
			Region:             "us-east-1",
			InstanceType:       "t3.micro",
			IdleTimeoutMinutes: 30,
		},
	}
	return s
}

// SetFyneApp sets the Fyne application instance
func (s *SettingsWindow) SetFyneApp(fyneApp fyne.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fyneApp = fyneApp
}

// GetFyneApp returns the Fyne application instance, creating one if needed
func (s *SettingsWindow) GetFyneApp() fyne.App {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fyneApp == nil {
		s.fyneApp = app.New()
	}
	return s.fyneApp
}

// SetOnStarted registers a callback to be called when the Fyne app has started.
// The callback receives the fyne.App instance and is stored until RunFyneLoop is called.
// This ensures the callback is registered immediately before the event loop starts,
// avoiding "tray not ready" errors that occur when the app is created too early.
func (s *SettingsWindow) SetOnStarted(callback func(fyne.App)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onStarted = callback
}

// Show displays the settings window.
// This method is safe to call from any goroutine - it schedules UI operations
// on the main Fyne thread to avoid thread safety issues.
func (s *SettingsWindow) Show() {
	// Get fyneApp reference under lock
	s.mu.Lock()
	if s.fyneApp == nil {
		s.fyneApp = app.New()
	}
	alreadyCreated := s.window != nil
	s.mu.Unlock()

	// Schedule all UI operations on the main Fyne thread
	fyne.DoAndWait(func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Double-check window creation in case another goroutine created it
		if !alreadyCreated && s.window == nil {
			s.createWindow()
		}

		s.loadConfigToForm()
		s.window.Show()
		s.visible = true
	})
}

// Hide hides the settings window.
// This method is safe to call from any goroutine.
func (s *SettingsWindow) Hide() {
	s.mu.Lock()
	window := s.window
	s.mu.Unlock()

	if window != nil {
		fyne.DoAndWait(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.window != nil {
				s.window.Hide()
				s.visible = false
			}
		})
	}
}

// SetOnSave sets the callback function called when settings are saved
func (s *SettingsWindow) SetOnSave(callback func(config SettingsConfig)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSave = callback
}

// LoadConfig loads configuration into the settings window
func (s *SettingsWindow) LoadConfig(config SettingsConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentConfig = config
	if s.window != nil {
		s.loadConfigToForm()
	}
}

// GetConfig returns the current configuration
func (s *SettingsWindow) GetConfig() SettingsConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentConfig
}

// IsVisible returns whether the settings window is currently visible
func (s *SettingsWindow) IsVisible() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visible
}

// createWindow creates the settings window and all its widgets
func (s *SettingsWindow) createWindow() {
	s.window = s.fyneApp.NewWindow("My Own VPN - Settings")
	s.window.Resize(fyne.NewSize(500, 450))
	s.window.SetFixedSize(true)
	s.window.CenterOnScreen()

	// Set close handler to hide instead of destroying
	s.window.SetCloseIntercept(func() {
		s.mu.Lock()
		s.visible = false
		s.mu.Unlock()
		s.window.Hide()
	})

	// Create all sections
	providerSection := s.createProviderSection()
	awsSection := s.createAWSCredentialsSection()
	hetznerSection := s.createHetznerCredentialsSection()
	prefsSection := s.createPreferencesSection()
	buttons := s.createButtons()

	// Main content container
	content := container.NewVBox(
		providerSection,
		widget.NewSeparator(),
		awsSection,
		hetznerSection,
		widget.NewSeparator(),
		prefsSection,
		widget.NewSeparator(),
		buttons,
	)

	// Wrap in padding
	paddedContent := container.NewPadded(content)
	s.window.SetContent(paddedContent)

	// Set initial provider visibility
	s.updateProviderVisibility()
}

// createProviderSection creates the provider selection section
func (s *SettingsWindow) createProviderSection() *fyne.Container {
	providerLabel := widget.NewLabel("Cloud Provider")
	s.providerSelect = widget.NewSelect([]string{ProviderAWS, ProviderHetzner}, func(selected string) {
		s.onProviderChanged(selected)
	})
	s.providerSelect.SetSelected(ProviderAWS)

	regionLabel := widget.NewLabel("Region")
	s.regionSelect = widget.NewSelect(AWSRegions, nil)
	s.regionSelect.SetSelected("us-east-1")

	instanceLabel := widget.NewLabel("Instance Type")
	s.instanceTypeSelect = widget.NewSelect(AWSInstanceTypes, nil)
	s.instanceTypeSelect.SetSelected("t3.micro")

	return container.NewVBox(
		widget.NewLabelWithStyle("Provider Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.New(layout.NewFormLayout(),
			providerLabel, s.providerSelect,
			regionLabel, s.regionSelect,
			instanceLabel, s.instanceTypeSelect,
		),
	)
}

// createAWSCredentialsSection creates the AWS credentials section
func (s *SettingsWindow) createAWSCredentialsSection() *fyne.Container {
	s.awsAccessKeyEntry = widget.NewEntry()
	s.awsAccessKeyEntry.SetPlaceHolder("AKIAIOSFODNN7EXAMPLE")

	s.awsSecretKeyEntry = widget.NewPasswordEntry()
	s.awsSecretKeyEntry.SetPlaceHolder("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	accessKeyLabel := widget.NewLabel("Access Key ID")
	secretKeyLabel := widget.NewLabel("Secret Access Key")

	s.awsContainer = container.NewVBox(
		widget.NewLabelWithStyle("AWS Credentials", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.New(layout.NewFormLayout(),
			accessKeyLabel, s.awsAccessKeyEntry,
			secretKeyLabel, s.awsSecretKeyEntry,
		),
	)

	return s.awsContainer
}

// createHetznerCredentialsSection creates the Hetzner credentials section
func (s *SettingsWindow) createHetznerCredentialsSection() *fyne.Container {
	s.hetznerTokenEntry = widget.NewPasswordEntry()
	s.hetznerTokenEntry.SetPlaceHolder("Enter your Hetzner API token")

	tokenLabel := widget.NewLabel("API Token")

	s.hetznerContainer = container.NewVBox(
		widget.NewLabelWithStyle("Hetzner Credentials", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.New(layout.NewFormLayout(),
			tokenLabel, s.hetznerTokenEntry,
		),
	)

	return s.hetznerContainer
}

// createPreferencesSection creates the preferences section
func (s *SettingsWindow) createPreferencesSection() *fyne.Container {
	s.idleTimeoutCheck = widget.NewCheck("Enable idle timeout", func(enabled bool) {
		s.idleTimeoutEntry.Hidden = !enabled
		if enabled {
			s.idleTimeoutEntry.Show()
		} else {
			s.idleTimeoutEntry.Hide()
		}
	})

	s.idleTimeoutEntry = widget.NewEntry()
	s.idleTimeoutEntry.SetPlaceHolder("30")
	s.idleTimeoutEntry.SetText("30")
	s.idleTimeoutEntry.Hide()

	timeoutLabel := widget.NewLabel("Timeout (minutes)")
	timeoutLabel.Hide()

	return container.NewVBox(
		widget.NewLabelWithStyle("Preferences", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		s.idleTimeoutCheck,
		container.NewHBox(
			widget.NewLabel("Timeout (minutes):"),
			s.idleTimeoutEntry,
		),
	)
}

// createButtons creates the Save and Cancel buttons
func (s *SettingsWindow) createButtons() *fyne.Container {
	saveBtn := widget.NewButton("Save", func() {
		s.onSaveClicked()
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancel", func() {
		s.onCancelClicked()
	})

	return container.NewHBox(
		layout.NewSpacer(),
		cancelBtn,
		saveBtn,
	)
}

// onProviderChanged handles provider selection changes
func (s *SettingsWindow) onProviderChanged(provider string) {
	s.mu.Lock()
	s.currentConfig.Provider = provider
	s.mu.Unlock()

	s.updateProviderVisibility()
	s.updateRegionsForProvider(provider)
	s.updateInstanceTypesForProvider(provider)
}

// updateProviderVisibility shows/hides credential sections based on provider
func (s *SettingsWindow) updateProviderVisibility() {
	provider := s.providerSelect.Selected

	if provider == ProviderAWS {
		s.awsContainer.Show()
		s.hetznerContainer.Hide()
	} else {
		s.awsContainer.Hide()
		s.hetznerContainer.Show()
	}
}

// updateRegionsForProvider updates the region dropdown for the selected provider
func (s *SettingsWindow) updateRegionsForProvider(provider string) {
	var regions []string
	var defaultRegion string

	if provider == ProviderAWS {
		regions = AWSRegions
		defaultRegion = "us-east-1"
	} else {
		regions = HetznerRegions
		defaultRegion = "nbg1"
	}

	s.regionSelect.Options = regions
	s.regionSelect.SetSelected(defaultRegion)
	s.regionSelect.Refresh()
}

// updateInstanceTypesForProvider updates the instance type dropdown for the selected provider
func (s *SettingsWindow) updateInstanceTypesForProvider(provider string) {
	var types []string
	var defaultType string

	if provider == ProviderAWS {
		types = AWSInstanceTypes
		defaultType = "t3.micro"
	} else {
		types = HetznerInstanceTypes
		defaultType = "cx22"
	}

	s.instanceTypeSelect.Options = types
	s.instanceTypeSelect.SetSelected(defaultType)
	s.instanceTypeSelect.Refresh()
}

// loadConfigToForm populates form fields from current config
func (s *SettingsWindow) loadConfigToForm() {
	if s.providerSelect != nil {
		s.providerSelect.SetSelected(s.currentConfig.Provider)
	}
	if s.regionSelect != nil {
		s.updateRegionsForProvider(s.currentConfig.Provider)
		s.regionSelect.SetSelected(s.currentConfig.Region)
	}
	if s.instanceTypeSelect != nil {
		s.updateInstanceTypesForProvider(s.currentConfig.Provider)
		s.instanceTypeSelect.SetSelected(s.currentConfig.InstanceType)
	}
	if s.awsAccessKeyEntry != nil {
		s.awsAccessKeyEntry.SetText(s.currentConfig.AWSAccessKey)
	}
	if s.awsSecretKeyEntry != nil {
		s.awsSecretKeyEntry.SetText(s.currentConfig.AWSSecretKey)
	}
	if s.hetznerTokenEntry != nil {
		s.hetznerTokenEntry.SetText(s.currentConfig.HetznerToken)
	}
	if s.idleTimeoutCheck != nil {
		s.idleTimeoutCheck.SetChecked(s.currentConfig.IdleTimeoutEnabled)
	}
	if s.idleTimeoutEntry != nil {
		if s.currentConfig.IdleTimeoutMinutes > 0 {
			s.idleTimeoutEntry.SetText(strconv.Itoa(s.currentConfig.IdleTimeoutMinutes))
		}
		if s.currentConfig.IdleTimeoutEnabled {
			s.idleTimeoutEntry.Show()
		} else {
			s.idleTimeoutEntry.Hide()
		}
	}

	s.updateProviderVisibility()
}

// collectConfigFromForm collects form data into a SettingsConfig
func (s *SettingsWindow) collectConfigFromForm() SettingsConfig {
	config := SettingsConfig{
		Provider:           s.providerSelect.Selected,
		Region:             s.regionSelect.Selected,
		InstanceType:       s.instanceTypeSelect.Selected,
		AWSAccessKey:       s.awsAccessKeyEntry.Text,
		AWSSecretKey:       s.awsSecretKeyEntry.Text,
		HetznerToken:       s.hetznerTokenEntry.Text,
		IdleTimeoutEnabled: s.idleTimeoutCheck.Checked,
	}

	if timeout, err := strconv.Atoi(s.idleTimeoutEntry.Text); err == nil && timeout > 0 {
		config.IdleTimeoutMinutes = timeout
	} else {
		config.IdleTimeoutMinutes = 30
	}

	return config
}

// onSaveClicked handles the Save button click
func (s *SettingsWindow) onSaveClicked() {
	config := s.collectConfigFromForm()
	errors := ValidateConfig(config)

	if len(errors) > 0 {
		// Show validation error dialog
		errDialog := widget.NewLabel("Please fix the following errors:\n\n" + joinErrors(errors))
		dialog := s.fyneApp.NewWindow("Validation Error")
		dialog.SetContent(container.NewPadded(container.NewVBox(
			errDialog,
			widget.NewButton("OK", func() {
				dialog.Close()
			}),
		)))
		dialog.Resize(fyne.NewSize(300, 150))
		dialog.CenterOnScreen()
		dialog.Show()
		return
	}

	s.mu.Lock()
	s.currentConfig = config
	callback := s.onSave
	s.mu.Unlock()

	if callback != nil {
		callback(config)
	}

	s.Hide()
}

// onCancelClicked handles the Cancel button click
func (s *SettingsWindow) onCancelClicked() {
	s.Hide()
}

// joinErrors joins error strings with newlines
func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "\n"
		}
		result += "• " + err
	}
	return result
}

// RunFyneLoop starts the Fyne event loop.
// This must be called from the main goroutine.
// The Fyne app is created here, immediately before starting the event loop,
// to avoid "tray not ready" errors that occur when there's a gap between
// app creation and event loop start.
func (s *SettingsWindow) RunFyneLoop() {
	s.mu.Lock()
	// Create the Fyne app right before running to minimize the gap
	// between app creation and event loop start
	if s.fyneApp == nil {
		s.fyneApp = app.New()
	}
	fyneApp := s.fyneApp
	onStartedCallback := s.onStarted
	s.mu.Unlock()

	// Register the OnStarted callback if one was provided
	if onStartedCallback != nil {
		fyneApp.Lifecycle().SetOnStarted(func() {
			onStartedCallback(fyneApp)
		})
	}

	// Run the Fyne event loop (blocks until Quit is called)
	fyneApp.Run()
}

// StopFyneLoop stops the Fyne event loop
func (s *SettingsWindow) StopFyneLoop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fyneApp != nil {
		s.fyneApp.Quit()
	}
}
