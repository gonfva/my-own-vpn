package app

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

const (
	appName = "My Own VPN"
)

// Notifier provides system notification capabilities.
type Notifier struct{}

// NewNotifier creates a new Notifier instance.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// ShowIdleWarning displays a notification warning that the VPN will disconnect
// due to inactivity.
func (n *Notifier) ShowIdleWarning(minutesRemaining int) error {
	title := appName
	var message string
	if minutesRemaining == 1 {
		message = "VPN will disconnect in 1 minute due to inactivity"
	} else {
		message = fmt.Sprintf("VPN will disconnect in %d minutes due to inactivity", minutesRemaining)
	}

	return beeep.Notify(title, message, "")
}

// ShowDisconnected displays a notification that the VPN has been disconnected.
func (n *Notifier) ShowDisconnected(reason string) error {
	title := appName
	message := "VPN disconnected"
	if reason != "" {
		message = fmt.Sprintf("VPN disconnected: %s", reason)
	}

	return beeep.Notify(title, message, "")
}

// ShowConnected displays a notification that the VPN is now connected.
func (n *Notifier) ShowConnected(serverIP string) error {
	title := appName
	message := fmt.Sprintf("Connected to VPN server %s", serverIP)

	return beeep.Notify(title, message, "")
}

// ShowError displays an error notification.
func (n *Notifier) ShowError(errMsg string) error {
	title := appName + " - Error"
	return beeep.Alert(title, errMsg, "")
}
