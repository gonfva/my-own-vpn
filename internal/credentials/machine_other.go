//go:build !darwin && !windows

package credentials

import (
	"os"
	"strings"
)

// getMachineIdentifier returns a stable machine-specific identifier on Linux and other platforms.
// It tries to read /etc/machine-id first, falling back to hostname+username if unavailable.
func getMachineIdentifier() string {
	// Try to read /etc/machine-id (standard on systemd-based Linux)
	data, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		machineID := strings.TrimSpace(string(data))
		if machineID != "" {
			return machineID
		}
	}

	// Try /var/lib/dbus/machine-id (older systems)
	data, err = os.ReadFile("/var/lib/dbus/machine-id")
	if err == nil {
		machineID := strings.TrimSpace(string(data))
		if machineID != "" {
			return machineID
		}
	}

	// Fallback to hostname + username
	return getFallbackIdentifier()
}

// getFallbackIdentifier returns a fallback identifier using hostname and username
func getFallbackIdentifier() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}
	return hostname + ":" + username
}
